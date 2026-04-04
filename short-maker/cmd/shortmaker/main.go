// cmd/shortmaker/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/west-garden/short-maker/internal/agent"
	"github.com/west-garden/short-maker/internal/domain"
	"github.com/west-garden/short-maker/internal/llm"
	"github.com/west-garden/short-maker/internal/store"
	"github.com/west-garden/short-maker/internal/strategy"
)

var rootCmd = &cobra.Command{
	Use:   "shortmaker",
	Short: "AI short drama production pipeline",
}

var runCmd = &cobra.Command{
	Use:   "run [script-file]",
	Short: "Run the production pipeline on a script file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		scriptPath := args[0]
		script, err := os.ReadFile(scriptPath)
		if err != nil {
			return fmt.Errorf("read script: %w", err)
		}

		styleName, _ := cmd.Flags().GetString("style")
		episodes, _ := cmd.Flags().GetInt("episodes")
		useMock, _ := cmd.Flags().GetBool("mock")
		llmModel, _ := cmd.Flags().GetString("model")
		dbPath, _ := cmd.Flags().GetString("db")
		strategyPath, _ := cmd.Flags().GetString("strategies")

		style := domain.Style(styleName)
		project := domain.NewProject(scriptPath, style, episodes)
		state := agent.NewPipelineState(project, string(script))

		agents, cleanup, err := buildAgents(useMock, llmModel, dbPath, strategyPath)
		if err != nil {
			return err
		}
		defer cleanup()

		checkpoint := func(phase agent.Phase, s *agent.PipelineState) error {
			log.Printf("  [checkpoint] phase %s completed", phase)
			return nil
		}

		orch := agent.NewOrchestrator(agents, checkpoint)
		result, err := orch.Run(context.Background(), state)
		if err != nil {
			return fmt.Errorf("pipeline failed: %w", err)
		}

		printSummary(result)
		return nil
	},
}

func buildAgents(useMock bool, llmModel, dbPath, strategyPath string) (map[agent.Phase]agent.Agent, func(), error) {
	agents := map[agent.Phase]agent.Agent{}
	cleanup := func() {}

	if useMock {
		// All mock agents
		for _, phase := range agent.DefaultFlow {
			p := phase
			agents[p] = agent.NewMockAgent(p, func(ctx context.Context, s *agent.PipelineState) (*agent.PipelineState, error) {
				log.Printf("  [mock-%s] processing...", p)
				return s, nil
			})
		}
		return agents, cleanup, nil
	}

	// Real agents for story understanding and character management
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, nil, fmt.Errorf("OPENAI_API_KEY environment variable is required when --mock=false")
	}
	baseURL := os.Getenv("OPENAI_BASE_URL")

	llmClient := llm.NewOpenAIClient(apiKey, baseURL)

	var st store.Store
	if dbPath != "" {
		sqliteStore, err := store.NewSQLiteStore(dbPath)
		if err != nil {
			return nil, nil, fmt.Errorf("open database: %w", err)
		}
		st = sqliteStore
		cleanup = func() { sqliteStore.Close() }
	}

	agents[agent.PhaseStoryUnderstanding] = agent.NewStoryAgent(llmClient, llmModel)
	agents[agent.PhaseCharacterAsset] = agent.NewCharacterAgent(llmClient, llmModel, st)

	// Storyboard agent with strategy engine
	if strategyPath != "" {
		repo, err := strategy.LoadFromFile(strategyPath)
		if err != nil {
			return nil, nil, fmt.Errorf("load strategies: %w", err)
		}
		agents[agent.PhaseStoryboard] = agent.NewStoryboardAgent(llmClient, llmModel, repo)
	} else {
		agents[agent.PhaseStoryboard] = agent.NewStoryboardAgent(llmClient, llmModel, nil)
	}

	// Remaining phases still use mocks — real implementations come in Plan 4+
	for _, phase := range agent.DefaultFlow {
		if _, ok := agents[phase]; !ok {
			p := phase
			agents[p] = agent.NewMockAgent(p, func(ctx context.Context, s *agent.PipelineState) (*agent.PipelineState, error) {
				log.Printf("  [mock-%s] processing...", p)
				return s, nil
			})
		}
	}

	return agents, cleanup, nil
}

func printSummary(result *agent.PipelineState) {
	log.Printf("=== Pipeline Complete ===")
	log.Printf("Project: %s", result.Project.Name)
	if result.Blueprint != nil {
		log.Printf("World: %s", result.Blueprint.WorldView)
		log.Printf("Characters: %d", len(result.Blueprint.Characters))
		for _, ch := range result.Blueprint.Characters {
			log.Printf("  - %s: %s", ch.Name, ch.Description)
		}
		log.Printf("Episodes: %d", len(result.Blueprint.Episodes))
	}
	log.Printf("Assets: %d", len(result.Assets))
	for _, a := range result.Assets {
		log.Printf("  - [%s] %s", a.Type, a.Name)
	}
	log.Printf("Storyboard shots: %d", len(result.Storyboard))
	log.Printf("Errors: %d", len(result.Errors))
}

func init() {
	runCmd.Flags().String("style", "manga", "Content style: manga, 3d, live_action")
	runCmd.Flags().Int("episodes", 10, "Number of episodes")
	runCmd.Flags().Bool("mock", true, "Use mock agents (set false for real LLM calls)")
	runCmd.Flags().String("model", "gpt-4o-mini", "LLM model name")
	runCmd.Flags().String("db", "", "SQLite database path (optional, enables persistence)")
	runCmd.Flags().String("strategies", "", "Path to strategies JSON file (enables real storyboard agent)")
	rootCmd.AddCommand(runCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
