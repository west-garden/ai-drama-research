// internal/router/mock_video.go
package router

import (
	"context"
	"os"
	"time"
)

// MockVideoAdapter creates a placeholder file for each video generation request.
type MockVideoAdapter struct{}

func NewMockVideoAdapter() *MockVideoAdapter {
	return &MockVideoAdapter{}
}

func (a *MockVideoAdapter) Name() string { return "mock-video" }

func (a *MockVideoAdapter) Capabilities() Capabilities {
	return Capabilities{
		Type:           ModelTypeVideo,
		Styles:         []string{"manga", "3d", "live_action"},
		MaxResolution:  "1024x1024",
		SupportsFusion: false,
	}
}

func (a *MockVideoAdapter) Generate(_ context.Context, req GenerateRequest) (*GenerateResponse, error) {
	start := time.Now()
	if err := os.WriteFile(req.OutputPath, []byte("mock-video-placeholder"), 0644); err != nil {
		return nil, err
	}
	return &GenerateResponse{
		FilePath:   req.OutputPath,
		ModelUsed:  "mock-video",
		Cost:       0.0,
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

func (a *MockVideoAdapter) HealthCheck(_ context.Context) error {
	return nil
}
