// internal/router/gemini_test.go
package router

import (
	"testing"
)

func TestGeminiImageAdapter_Name(t *testing.T) {
	a := NewGeminiImageAdapter("fake-key", "imagen-4.0-generate-001", "")
	if a.Name() != "gemini-image" {
		t.Errorf("Name() = %q, want %q", a.Name(), "gemini-image")
	}
}

func TestGeminiImageAdapter_Capabilities(t *testing.T) {
	a := NewGeminiImageAdapter("fake-key", "imagen-4.0-generate-001", "")
	caps := a.Capabilities()
	if caps.Type != ModelTypeImage {
		t.Errorf("Type = %q, want %q", caps.Type, ModelTypeImage)
	}
	if len(caps.Styles) != 3 {
		t.Errorf("len(Styles) = %d, want 3", len(caps.Styles))
	}
}

func TestGeminiVideoAdapter_Name(t *testing.T) {
	a := NewGeminiVideoAdapter("fake-key", "veo-3.1-generate-preview", "")
	if a.Name() != "gemini-video" {
		t.Errorf("Name() = %q, want %q", a.Name(), "gemini-video")
	}
}

func TestGeminiVideoAdapter_Capabilities(t *testing.T) {
	a := NewGeminiVideoAdapter("fake-key", "veo-3.1-generate-preview", "")
	caps := a.Capabilities()
	if caps.Type != ModelTypeVideo {
		t.Errorf("Type = %q, want %q", caps.Type, ModelTypeVideo)
	}
	if len(caps.Styles) != 3 {
		t.Errorf("len(Styles) = %d, want 3", len(caps.Styles))
	}
}
