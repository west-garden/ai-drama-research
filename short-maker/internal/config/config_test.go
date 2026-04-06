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
	if len(cfg.Image.Providers) != 1 || cfg.Image.Providers[0] != "gemini" {
		t.Errorf("Image.Providers = %v, want [gemini]", cfg.Image.Providers)
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
	if len(cfg.Image.Providers) != 1 || cfg.Image.Providers[0] != "mock" {
		t.Errorf("Image.Providers default = %v, want [mock]", cfg.Image.Providers)
	}
	if cfg.Video.Provider != "mock" {
		t.Errorf("Video.Provider default = %q, want %q", cfg.Video.Provider, "mock")
	}
	if len(cfg.Video.Providers) != 1 || cfg.Video.Providers[0] != "mock" {
		t.Errorf("Video.Providers default = %v, want [mock]", cfg.Video.Providers)
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

func TestLoad_Providers_NewFormat(t *testing.T) {
	yaml := `
image:
  providers: [gemini, jimeng]
  gemini:
    api_key: AIza-test
    model: imagen-4.0-generate-001
  jimeng:
    access_key: AK-test
    secret_key: SK-test
    req_key: jimeng_t2i_v40
video:
  providers: [jimeng]
  jimeng:
    access_key: AK-test
    secret_key: SK-test
    req_key: jimeng_vgfm_i2v_l20
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Image.Providers) != 2 {
		t.Fatalf("Image.Providers length = %d, want 2", len(cfg.Image.Providers))
	}
	if cfg.Image.Providers[0] != "gemini" || cfg.Image.Providers[1] != "jimeng" {
		t.Errorf("Image.Providers = %v, want [gemini jimeng]", cfg.Image.Providers)
	}
	// Provider should be first element
	if cfg.Image.Provider != "gemini" {
		t.Errorf("Image.Provider = %q, want %q", cfg.Image.Provider, "gemini")
	}
	if len(cfg.Video.Providers) != 1 || cfg.Video.Providers[0] != "jimeng" {
		t.Errorf("Video.Providers = %v, want [jimeng]", cfg.Video.Providers)
	}
}

func TestLoad_Providers_BackwardCompat(t *testing.T) {
	yaml := `
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
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// Old provider: field should be normalized into providers: list
	if len(cfg.Image.Providers) != 1 || cfg.Image.Providers[0] != "gemini" {
		t.Errorf("Image.Providers = %v, want [gemini]", cfg.Image.Providers)
	}
	if len(cfg.Video.Providers) != 1 || cfg.Video.Providers[0] != "jimeng" {
		t.Errorf("Video.Providers = %v, want [jimeng]", cfg.Video.Providers)
	}
}

func TestLoad_SharedProviders(t *testing.T) {
	yaml := `
providers:
  gemini:
    api_key: AIza-shared
    proxy: http://proxy:8080
  jimeng:
    access_key: AK-shared
    secret_key: SK-shared
image:
  providers: [gemini, jimeng]
  gemini:
    model: imagen-4.0-generate-001
  jimeng:
    req_key: jimeng_t2i_v40
video:
  providers: [gemini]
  gemini:
    model: veo-3.1-lite-generate-preview
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Image.Gemini.APIKey != "AIza-shared" {
		t.Errorf("Image.Gemini.APIKey = %q, want %q", cfg.Image.Gemini.APIKey, "AIza-shared")
	}
	if cfg.Image.Gemini.Proxy != "http://proxy:8080" {
		t.Errorf("Image.Gemini.Proxy = %q, want %q", cfg.Image.Gemini.Proxy, "http://proxy:8080")
	}
	if cfg.Image.Jimeng.AccessKey != "AK-shared" {
		t.Errorf("Image.Jimeng.AccessKey = %q, want %q", cfg.Image.Jimeng.AccessKey, "AK-shared")
	}
	if cfg.Image.Jimeng.SecretKey != "SK-shared" {
		t.Errorf("Image.Jimeng.SecretKey = %q, want %q", cfg.Image.Jimeng.SecretKey, "SK-shared")
	}
	if cfg.Video.Gemini.APIKey != "AIza-shared" {
		t.Errorf("Video.Gemini.APIKey = %q, want %q", cfg.Video.Gemini.APIKey, "AIza-shared")
	}
	if cfg.Video.Gemini.Proxy != "http://proxy:8080" {
		t.Errorf("Video.Gemini.Proxy = %q, want %q", cfg.Video.Gemini.Proxy, "http://proxy:8080")
	}
}

func TestLoad_InlineOverridesProvider(t *testing.T) {
	yaml := `
providers:
  gemini:
    api_key: AIza-shared
    proxy: http://proxy:8080
image:
  providers: [gemini]
  gemini:
    api_key: AIza-inline
    model: imagen-4.0-generate-001
    proxy: http://inline-proxy:9090
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Image.Gemini.APIKey != "AIza-inline" {
		t.Errorf("Image.Gemini.APIKey = %q, want %q (inline should override provider)", cfg.Image.Gemini.APIKey, "AIza-inline")
	}
	if cfg.Image.Gemini.Proxy != "http://inline-proxy:9090" {
		t.Errorf("Image.Gemini.Proxy = %q, want %q (inline should override provider)", cfg.Image.Gemini.Proxy, "http://inline-proxy:9090")
	}
}

func TestLoad_LLMInheritsFromProvider(t *testing.T) {
	yaml := `
providers:
  gemini:
    api_key: AIza-shared
    proxy: http://proxy:8080
llm:
  provider: gemini
  base_url: https://generativelanguage.googleapis.com/v1beta/openai
  model: gemini-2.5-flash
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LLM.APIKey != "AIza-shared" {
		t.Errorf("LLM.APIKey = %q, want %q", cfg.LLM.APIKey, "AIza-shared")
	}
	if cfg.LLM.Proxy != "http://proxy:8080" {
		t.Errorf("LLM.Proxy = %q, want %q", cfg.LLM.Proxy, "http://proxy:8080")
	}
}

func TestLoad_Providers_ValidationAll(t *testing.T) {
	// Both providers listed but jimeng missing secret_key
	yaml := `
image:
  providers: [gemini, jimeng]
  gemini:
    api_key: AIza-test
  jimeng:
    access_key: AK-test
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected validation error when jimeng in providers list is missing secret_key")
	}
}
