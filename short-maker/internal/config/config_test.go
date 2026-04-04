package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_FullConfig(t *testing.T) {
	yaml := `
llm:
  provider: openai
  api_key: sk-test
  base_url: https://api.example.com/v1
  model: gpt-4o
image:
  provider: gemini
  gemini:
    api_key: AIza-test
    model: imagen-4.0-generate-001
video:
  provider: jimeng
  jimeng:
    access_key: AK-test
    secret_key: SK-test
    req_key: jimeng_vgfm_i2v_l20
output_dir: ./out
db_path: ./test.db
strategies_path: ./strats.json
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LLM.APIKey != "sk-test" {
		t.Errorf("LLM.APIKey = %q, want %q", cfg.LLM.APIKey, "sk-test")
	}
	if cfg.Image.Provider != "gemini" {
		t.Errorf("Image.Provider = %q, want %q", cfg.Image.Provider, "gemini")
	}
	if cfg.Video.Jimeng.AccessKey != "AK-test" {
		t.Errorf("Video.Jimeng.AccessKey = %q, want %q", cfg.Video.Jimeng.AccessKey, "AK-test")
	}
	if cfg.OutputDir != "./out" {
		t.Errorf("OutputDir = %q, want %q", cfg.OutputDir, "./out")
	}
}

func TestLoad_Defaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte("{}"), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LLM.Provider != "openai" {
		t.Errorf("LLM.Provider default = %q, want %q", cfg.LLM.Provider, "openai")
	}
	if cfg.LLM.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("LLM.BaseURL default = %q", cfg.LLM.BaseURL)
	}
	if cfg.LLM.Model != "gpt-4o-mini" {
		t.Errorf("LLM.Model default = %q", cfg.LLM.Model)
	}
	if cfg.Image.Provider != "mock" {
		t.Errorf("Image.Provider default = %q, want %q", cfg.Image.Provider, "mock")
	}
	if cfg.Video.Provider != "mock" {
		t.Errorf("Video.Provider default = %q, want %q", cfg.Video.Provider, "mock")
	}
	if cfg.OutputDir != "./output" {
		t.Errorf("OutputDir default = %q", cfg.OutputDir)
	}
	if cfg.DBPath != "./shortmaker.db" {
		t.Errorf("DBPath default = %q", cfg.DBPath)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	cfg, err := Load("/nonexistent/config.yaml")
	if err != nil {
		t.Fatalf("Load should not error for missing file, got: %v", err)
	}
	// Should return all-mock defaults
	if cfg.Image.Provider != "mock" {
		t.Errorf("Image.Provider = %q, want mock", cfg.Image.Provider)
	}
}

func TestLoad_ValidationError_GeminiNoKey(t *testing.T) {
	yaml := `
image:
  provider: gemini
  gemini:
    model: imagen-4.0-generate-001
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected validation error for gemini without api_key")
	}
}

func TestLoad_ValidationError_JimengNoKey(t *testing.T) {
	yaml := `
video:
  provider: jimeng
  jimeng:
    access_key: AK-test
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected validation error for jimeng without secret_key")
	}
}
