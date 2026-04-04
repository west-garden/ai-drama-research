package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	LLM            LLMConfig   `yaml:"llm"`
	Image          ImageConfig `yaml:"image"`
	Video          VideoConfig `yaml:"video"`
	OutputDir      string      `yaml:"output_dir"`
	DBPath         string      `yaml:"db_path"`
	StrategiesPath string      `yaml:"strategies_path"`
}

type LLMConfig struct {
	Provider string `yaml:"provider"`
	APIKey   string `yaml:"api_key"`
	BaseURL  string `yaml:"base_url"`
	Model    string `yaml:"model"`
}

type ImageConfig struct {
	Provider string       `yaml:"provider"`
	Gemini   GeminiConfig `yaml:"gemini"`
	Jimeng   JimengConfig `yaml:"jimeng"`
}

type VideoConfig struct {
	Provider string       `yaml:"provider"`
	Gemini   GeminiConfig `yaml:"gemini"`
	Jimeng   JimengConfig `yaml:"jimeng"`
}

type GeminiConfig struct {
	APIKey string `yaml:"api_key"`
	Model  string `yaml:"model"`
}

type JimengConfig struct {
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	ReqKey    string `yaml:"req_key"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			applyDefaults(cfg)
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	applyDefaults(cfg)

	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.LLM.Provider == "" {
		cfg.LLM.Provider = "openai"
	}
	if cfg.LLM.BaseURL == "" {
		cfg.LLM.BaseURL = "https://api.openai.com/v1"
	}
	if cfg.LLM.Model == "" {
		cfg.LLM.Model = "gpt-4o-mini"
	}
	if cfg.Image.Provider == "" {
		cfg.Image.Provider = "mock"
	}
	if cfg.Video.Provider == "" {
		cfg.Video.Provider = "mock"
	}
	if cfg.OutputDir == "" {
		cfg.OutputDir = "./output"
	}
	if cfg.DBPath == "" {
		cfg.DBPath = "./shortmaker.db"
	}
}

func validate(cfg *Config) error {
	if cfg.Image.Provider == "gemini" && cfg.Image.Gemini.APIKey == "" {
		return fmt.Errorf("image.gemini.api_key is required when image.provider=gemini")
	}
	if cfg.Image.Provider == "jimeng" {
		if cfg.Image.Jimeng.AccessKey == "" || cfg.Image.Jimeng.SecretKey == "" {
			return fmt.Errorf("image.jimeng.access_key and secret_key are required when image.provider=jimeng")
		}
	}
	if cfg.Video.Provider == "gemini" && cfg.Video.Gemini.APIKey == "" {
		return fmt.Errorf("video.gemini.api_key is required when video.provider=gemini")
	}
	if cfg.Video.Provider == "jimeng" {
		if cfg.Video.Jimeng.AccessKey == "" || cfg.Video.Jimeng.SecretKey == "" {
			return fmt.Errorf("video.jimeng.access_key and secret_key are required when video.provider=jimeng")
		}
	}
	return nil
}
