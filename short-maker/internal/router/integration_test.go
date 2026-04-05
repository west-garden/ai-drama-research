//go:build integration

package router_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/west-garden/short-maker/internal/config"
	"github.com/west-garden/short-maker/internal/router"
)

// buildImageAdapter creates the image adapter for a given provider name,
// returning nil if the provider's credentials are not configured.
func buildImageAdapter(cfg *config.Config, provider string) router.ModelAdapter {
	switch provider {
	case "gemini":
		if cfg.Image.Gemini.APIKey == "" {
			return nil
		}
		return router.NewGeminiImageAdapter(cfg.Image.Gemini.APIKey, cfg.Image.Gemini.Model, cfg.Image.Gemini.Proxy)
	case "jimeng":
		if cfg.Image.Jimeng.AccessKey == "" || cfg.Image.Jimeng.SecretKey == "" {
			return nil
		}
		return router.NewJimengImageAdapter(cfg.Image.Jimeng.AccessKey, cfg.Image.Jimeng.SecretKey, cfg.Image.Jimeng.ReqKey)
	default:
		return nil
	}
}

func TestImageAdapter_Integration(t *testing.T) {
	cfg, err := config.Load("../../config.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if len(cfg.Image.Providers) == 0 {
		t.Skip("no image providers configured")
	}

	for _, provider := range cfg.Image.Providers {
		t.Run(provider, func(t *testing.T) {
			adapter := buildImageAdapter(cfg, provider)
			if adapter == nil {
				t.Skipf("%s image provider credentials not configured, skipping", provider)
				return
			}

			outDir := t.TempDir()
			outPath := filepath.Join(outDir, "test_output.png")

			resp, err := adapter.Generate(context.Background(), router.GenerateRequest{
				Prompt:     "A cute orange cat sitting on a wooden fence at sunset, manga style",
				Style:      "manga",
				OutputPath: outPath,
			})
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			info, err := os.Stat(resp.FilePath)
			if err != nil {
				t.Fatalf("output file not found: %v", err)
			}
			if info.Size() == 0 {
				t.Fatal("output file is empty")
			}

			t.Logf("Success! [%s] Image generated:", provider)
			t.Logf("  File: %s", resp.FilePath)
			t.Logf("  Size: %d bytes", info.Size())
			t.Logf("  Model: %s", resp.ModelUsed)
			t.Logf("  Duration: %dms", resp.DurationMs)
		})
	}
}
