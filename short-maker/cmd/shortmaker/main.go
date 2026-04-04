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
	"github.com/west-garden/short-maker/internal/config"
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
		configPath, _ := cmd.Flags().GetString("config")

		cfg, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		style := domain.Style(styleName)
		project := domain.NewProject(scriptPath, style, episodes)
		state := agent.NewPipelineState(project, string(script))

		agents, cleanup, err := buildAgents(cfg)
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
		configPath, _ := cmd.Flags().GetString("config")

		cfg, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		agents, cleanup, err := buildAgents(cfg)
		if err != nil {
			return err
		}
		defer cleanup()

		st, err := store.NewSQLiteStore(cfg.DBPath)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer st.Close()

		st.RecoverRunningPipelines(context.Background())

		srv := api.NewServer(agents, st, cfg.OutputDir)
		log.Printf("Starting web console at http://localhost:%d", port)
		return srv.Start(port)
	},
}

func buildAgents(cfg *config.Config) (map[agent.Phase]agent.Agent, func(), error) {
	agents := map[agent.Phase]agent.Agent{}
	cleanup := func() {}

	// LLM client — api_key from config, fallback to env
	apiKey := cfg.LLM.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if apiKey == "" {
		return nil, nil, fmt.Errorf("LLM API key required: set llm.api_key in config or OPENAI_API_KEY env")
	}
	llmClient := llm.NewOpenAIClient(apiKey, cfg.LLM.BaseURL)

	// Store
	var st store.Store
	if cfg.DBPath != "" {
		sqliteStore, err := store.NewSQLiteStore(cfg.DBPath)
		if err != nil {
			return nil, nil, fmt.Errorf("open database: %w", err)
		}
		st = sqliteStore
		cleanup = func() { sqliteStore.Close() }
	}

	// Story + Character agents (always use real LLM)
	agents[agent.PhaseStoryUnderstanding] = agent.NewStoryAgent(llmClient, cfg.LLM.Model)
	agents[agent.PhaseCharacterAsset] = agent.NewCharacterAgent(llmClient, cfg.LLM.Model, st)

	// Storyboard agent
	if cfg.StrategiesPath != "" {
		repo, err := strategy.LoadFromFile(cfg.StrategiesPath)
		if err != nil {
			return nil, nil, fmt.Errorf("load strategies: %w", err)
		}
		agents[agent.PhaseStoryboard] = agent.NewStoryboardAgent(llmClient, cfg.LLM.Model, repo)
	} else {
		agents[agent.PhaseStoryboard] = agent.NewStoryboardAgent(llmClient, cfg.LLM.Model, nil)
	}

	// Image adapter
	var imageAdapter router.ModelAdapter
	switch cfg.Image.Provider {
	case "gemini":
		imageAdapter = router.NewGeminiImageAdapter(cfg.Image.Gemini.APIKey, cfg.Image.Gemini.Model)
	case "jimeng":
		imageAdapter = router.NewJimengImageAdapter(cfg.Image.Jimeng.AccessKey, cfg.Image.Jimeng.SecretKey, cfg.Image.Jimeng.ReqKey)
	default:
		imageAdapter = router.NewMockImageAdapter()
	}

	// Video adapter
	var videoAdapter router.ModelAdapter
	switch cfg.Video.Provider {
	case "gemini":
		videoAdapter = router.NewGeminiVideoAdapter(cfg.Video.Gemini.APIKey, cfg.Video.Gemini.Model)
	case "jimeng":
		videoAdapter = router.NewJimengVideoAdapter(cfg.Video.Jimeng.AccessKey, cfg.Video.Jimeng.SecretKey, cfg.Video.Jimeng.ReqKey)
	default:
		videoAdapter = router.NewMockVideoAdapter()
	}

	modelRouter := router.NewModelRouter(imageAdapter, videoAdapter)
	checker := quality.NewMockChecker()

	agents[agent.PhaseImageGeneration] = agent.NewImageGenAgent(modelRouter, checker, cfg.OutputDir)
	agents[agent.PhaseVideoGeneration] = agent.NewVideoGenAgent(modelRouter, checker, cfg.OutputDir)

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
	runCmd.Flags().String("config", "./config.yaml", "Path to config file")
	runCmd.Flags().String("style", "manga", "Content style: manga, 3d, live_action")
	runCmd.Flags().Int("episodes", 10, "Number of episodes")
	rootCmd.AddCommand(runCmd)

	serveCmd.Flags().String("config", "./config.yaml", "Path to config file")
	serveCmd.Flags().Int("port", 8080, "HTTP server port")
	rootCmd.AddCommand(serveCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
