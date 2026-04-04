package router

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"google.golang.org/genai"
)

// GeminiImageAdapter uses Imagen via the Google GenAI SDK.
type GeminiImageAdapter struct {
	apiKey string
	model  string
}

func NewGeminiImageAdapter(apiKey, model string) *GeminiImageAdapter {
	return &GeminiImageAdapter{apiKey: apiKey, model: model}
}

func (a *GeminiImageAdapter) Name() string { return "gemini-image" }

func (a *GeminiImageAdapter) Capabilities() Capabilities {
	return Capabilities{
		Type:           ModelTypeImage,
		Styles:         []string{"manga", "3d", "live_action"},
		MaxResolution:  "1024x1024",
		SupportsFusion: false,
	}
}

func (a *GeminiImageAdapter) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	start := time.Now()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  a.apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("gemini client: %w", err)
	}

	config := &genai.GenerateImagesConfig{
		NumberOfImages: 1,
		AspectRatio:    "9:16",
	}

	response, err := client.Models.GenerateImages(ctx, a.model, req.Prompt, config)
	if err != nil {
		return nil, fmt.Errorf("gemini generate image: %w", err)
	}

	if len(response.GeneratedImages) == 0 {
		return nil, fmt.Errorf("gemini returned no images")
	}

	imageBytes := response.GeneratedImages[0].Image.ImageBytes
	if err := os.WriteFile(req.OutputPath, imageBytes, 0644); err != nil {
		return nil, fmt.Errorf("write image: %w", err)
	}

	return &GenerateResponse{
		FilePath:   req.OutputPath,
		ModelUsed:  a.model,
		Cost:       0.0,
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

func (a *GeminiImageAdapter) HealthCheck(ctx context.Context) error {
	_, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  a.apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	return err
}

// GeminiVideoAdapter uses Veo via the Google GenAI SDK.
type GeminiVideoAdapter struct {
	apiKey string
	model  string
}

func NewGeminiVideoAdapter(apiKey, model string) *GeminiVideoAdapter {
	return &GeminiVideoAdapter{apiKey: apiKey, model: model}
}

func (a *GeminiVideoAdapter) Name() string { return "gemini-video" }

func (a *GeminiVideoAdapter) Capabilities() Capabilities {
	return Capabilities{
		Type:           ModelTypeVideo,
		Styles:         []string{"manga", "3d", "live_action"},
		MaxResolution:  "1280x720",
		SupportsFusion: false,
	}
}

func (a *GeminiVideoAdapter) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	start := time.Now()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  a.apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("gemini client: %w", err)
	}

	// Build image input if source image provided (image-to-video).
	var image *genai.Image
	if req.SourceImage != "" {
		imgData, err := os.ReadFile(req.SourceImage)
		if err != nil {
			return nil, fmt.Errorf("read source image: %w", err)
		}
		image = &genai.Image{
			ImageBytes: imgData,
			MIMEType:   "image/png",
		}
	}

	config := &genai.GenerateVideosConfig{
		AspectRatio: "9:16",
	}

	operation, err := client.Models.GenerateVideos(ctx, a.model, req.Prompt, image, config)
	if err != nil {
		return nil, fmt.Errorf("gemini generate video: %w", err)
	}

	// Poll until done — max 5 minutes.
	const pollInterval = 10 * time.Second
	const maxWait = 5 * time.Minute
	deadline := time.Now().Add(maxWait)

	for !operation.Done {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("gemini video generation timed out after %v", maxWait)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}
		log.Printf("[gemini-video] polling operation status...")
		operation, err = client.Operations.GetVideosOperation(ctx, operation, nil)
		if err != nil {
			return nil, fmt.Errorf("gemini poll video: %w", err)
		}
	}

	// Download the generated video.
	if operation.Response == nil || len(operation.Response.GeneratedVideos) == 0 {
		return nil, fmt.Errorf("gemini returned no videos")
	}

	video := operation.Response.GeneratedVideos[0]
	uri := genai.NewDownloadURIFromGeneratedVideo(video)
	videoData, err := client.Files.Download(ctx, uri, nil)
	if err != nil {
		return nil, fmt.Errorf("gemini download video: %w", err)
	}

	if err := os.WriteFile(req.OutputPath, videoData, 0644); err != nil {
		return nil, fmt.Errorf("write video: %w", err)
	}

	return &GenerateResponse{
		FilePath:   req.OutputPath,
		ModelUsed:  a.model,
		Cost:       0.0,
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

func (a *GeminiVideoAdapter) HealthCheck(ctx context.Context) error {
	_, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  a.apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	return err
}
