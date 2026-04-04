package router

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"time"
)

// MockImageAdapter creates a 1x1 PNG placeholder for each generation request.
type MockImageAdapter struct{}

func NewMockImageAdapter() *MockImageAdapter {
	return &MockImageAdapter{}
}

func (a *MockImageAdapter) Name() string { return "mock-image" }

func (a *MockImageAdapter) Capabilities() Capabilities {
	return Capabilities{
		Type:           ModelTypeImage,
		Styles:         []string{"manga", "3d", "live_action"},
		MaxResolution:  "1024x1024",
		SupportsFusion: false,
	}
}

func (a *MockImageAdapter) Generate(_ context.Context, req GenerateRequest) (*GenerateResponse, error) {
	start := time.Now()

	f, err := os.Create(req.OutputPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.White)
	if err := png.Encode(f, img); err != nil {
		return nil, err
	}

	return &GenerateResponse{
		FilePath:   req.OutputPath,
		ModelUsed:  "mock-image",
		Cost:       0.0,
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

func (a *MockImageAdapter) HealthCheck(_ context.Context) error {
	return nil
}
