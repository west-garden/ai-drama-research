// cmd/shortmaker/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/west-garden/short-maker/internal/agent"
	"github.com/west-garden/short-maker/internal/api"
	"github.com/west-garden/short-maker/internal/domain"
	"github.com/west-garden/short-maker/internal/llm"
	"github.com/west-garden/short-maker/internal/quality"
	"github.com/west-garden/short-maker/internal/router"
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
		outputDir, _ := cmd.Flags().GetString("output")

		style := domain.Style(styleName)
		project := domain.NewProject(scriptPath, style, episodes)
		state := agent.NewPipelineState(project, string(script))

		agents, cleanup, err := buildAgents(useMock, llmModel, dbPath, strategyPath, outputDir)
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

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web console server",
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		outputDir, _ := cmd.Flags().GetString("output")
		dbPath, _ := cmd.Flags().GetString("db")
		useMock, _ := cmd.Flags().GetBool("mock")
		llmModel, _ := cmd.Flags().GetString("model")
		strategyPath, _ := cmd.Flags().GetString("strategies")

		if dbPath == "" {
			dbPath = "./shortmaker.db"
		}

		agents, cleanup, err := buildAgents(useMock, llmModel, dbPath, strategyPath, outputDir)
		if err != nil {
			return err
		}
		defer cleanup()

		st, err := store.NewSQLiteStore(dbPath)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer st.Close()

		// Mark any previously running pipelines as failed
		st.RecoverRunningPipelines(context.Background())

		srv := api.NewServer(agents, st, outputDir)
		log.Printf("Starting web console at http://localhost:%d", port)
		return srv.Start(port)
	},
}

func buildAgents(useMock bool, llmModel, dbPath, strategyPath, outputDir string) (map[agent.Phase]agent.Agent, func(), error) {
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

	// Image and video generation agents with model router
	imageAdapter := router.NewMockImageAdapter()
	videoAdapter := router.NewMockVideoAdapter()
	modelRouter := router.NewModelRouter(imageAdapter, videoAdapter)
	checker := quality.NewMockChecker()

	agents[agent.PhaseImageGeneration] = agent.NewImageGenAgent(modelRouter, checker, outputDir)
	agents[agent.PhaseVideoGeneration] = agent.NewVideoGenAgent(modelRouter, checker, outputDir)

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
	log.Printf("Generated images: %d", len(result.Images))
	for _, img := range result.Images {
		log.Printf("  - ep%02d/shot%03d [%s] score:%d %s", img.EpisodeNum, img.ShotNumber, img.Grade, img.ImageScore, img.ImagePath)
	}
	log.Printf("Generated videos: %d", len(result.Videos))
	for _, vid := range result.Videos {
		log.Printf("  - ep%02d/shot%03d [%s] score:%d %s", vid.EpisodeNum, vid.ShotNumber, vid.Grade, vid.VideoScore, vid.VideoPath)
	}
	log.Printf("Errors: %d", len(result.Errors))
}

func init() {
	runCmd.Flags().String("style", "manga", "Content style: manga, 3d, live_action")
	runCmd.Flags().Int("episodes", 10, "Number of episodes")
	runCmd.Flags().Bool("mock", true, "Use mock agents (set false for real LLM calls)")
	runCmd.Flags().String("model", "gpt-4o-mini", "LLM model name")
	runCmd.Flags().String("db", "", "SQLite database path (optional, enables persistence)")
	runCmd.Flags().String("strategies", "", "Path to strategies JSON file (enables real storyboard agent)")
	runCmd.Flags().String("output", "./output", "Output directory for generated files")
	rootCmd.AddCommand(runCmd)

	serveCmd.Flags().Int("port", 8080, "HTTP server port")
	serveCmd.Flags().String("output", "./output", "Output directory for generated files")
	serveCmd.Flags().String("db", "./shortmaker.db", "SQLite database path")
	serveCmd.Flags().Bool("mock", true, "Use mock agents")
	serveCmd.Flags().String("model", "gpt-4o-mini", "LLM model name")
	serveCmd.Flags().String("strategies", "", "Path to strategies JSON file")
	rootCmd.AddCommand(serveCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
