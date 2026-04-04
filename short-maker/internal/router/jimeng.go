package router

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/volcengine/volc-sdk-golang/service/visual"
)

// JimengImageAdapter uses Volcengine Visual API for text-to-image.
type JimengImageAdapter struct {
	ak     string
	sk     string
	reqKey string
}

func NewJimengImageAdapter(ak, sk, reqKey string) *JimengImageAdapter {
	return &JimengImageAdapter{ak: ak, sk: sk, reqKey: reqKey}
}

func (a *JimengImageAdapter) Name() string { return "jimeng-image" }

func (a *JimengImageAdapter) Capabilities() Capabilities {
	return Capabilities{
		Type:           ModelTypeImage,
		Styles:         []string{"manga", "3d", "live_action"},
		MaxResolution:  "1024x1024",
		SupportsFusion: false,
	}
}

func (a *JimengImageAdapter) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	start := time.Now()

	instance := visual.NewInstance()
	instance.Client.SetAccessKey(a.ak)
	instance.Client.SetSecretKey(a.sk)

	params := map[string]interface{}{
		"req_key":    a.reqKey,
		"prompt":     req.Prompt,
		"width":      1024,
		"height":     1024,
		"return_url": true,
	}

	resp, _, err := instance.CVProcess(params)
	if err != nil {
		return nil, fmt.Errorf("jimeng image: %w", err)
	}

	// Navigate the response map to get image URLs.
	// Response structure: {"data": {"image_urls": [...]}}
	imageURLs, err := extractStringSlice(resp, "data", "image_urls")
	if err != nil {
		return nil, fmt.Errorf("jimeng parse response: %w", err)
	}
	if len(imageURLs) == 0 {
		return nil, fmt.Errorf("jimeng returned no image URLs")
	}

	// Download the image.
	if err := downloadFile(ctx, imageURLs[0], req.OutputPath); err != nil {
		return nil, fmt.Errorf("jimeng download image: %w", err)
	}

	return &GenerateResponse{
		FilePath:   req.OutputPath,
		ModelUsed:  a.reqKey,
		Cost:       0.0,
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

func (a *JimengImageAdapter) HealthCheck(_ context.Context) error {
	return nil
}

// JimengVideoAdapter uses Volcengine Visual API for image-to-video.
type JimengVideoAdapter struct {
	ak     string
	sk     string
	reqKey string
}

func NewJimengVideoAdapter(ak, sk, reqKey string) *JimengVideoAdapter {
	return &JimengVideoAdapter{ak: ak, sk: sk, reqKey: reqKey}
}

func (a *JimengVideoAdapter) Name() string { return "jimeng-video" }

func (a *JimengVideoAdapter) Capabilities() Capabilities {
	return Capabilities{
		Type:           ModelTypeVideo,
		Styles:         []string{"manga", "3d", "live_action"},
		MaxResolution:  "1280x720",
		SupportsFusion: false,
	}
}

func (a *JimengVideoAdapter) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	start := time.Now()

	instance := visual.NewInstance()
	instance.Client.SetAccessKey(a.ak)
	instance.Client.SetSecretKey(a.sk)

	params := map[string]interface{}{
		"req_key": a.reqKey,
		"prompt":  req.Prompt,
	}
	if req.SourceImage != "" {
		params["image_urls"] = []string{req.SourceImage}
	}

	// Submit async task.
	submitResp, _, err := instance.CVSync2AsyncSubmitTask(params)
	if err != nil {
		return nil, fmt.Errorf("jimeng submit video: %w", err)
	}

	taskID, err := extractString(submitResp, "data", "task_id")
	if err != nil {
		return nil, fmt.Errorf("jimeng parse submit: %w", err)
	}

	// Poll until done -- max 5 minutes.
	const pollInterval = 10 * time.Second
	const maxWait = 5 * time.Minute
	deadline := time.Now().Add(maxWait)

	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("jimeng video generation timed out after %v", maxWait)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}

		log.Printf("[jimeng-video] polling task %s...", taskID)

		pollParams := map[string]interface{}{
			"req_key": a.reqKey,
			"task_id": taskID,
		}
		pollResp, _, err := instance.CVSync2AsyncGetResult(pollParams)
		if err != nil {
			return nil, fmt.Errorf("jimeng poll video: %w", err)
		}

		status, _ := extractString(pollResp, "data", "status")
		if status == "done" {
			videoURLs, err := extractStringSlice(pollResp, "data", "video_urls")
			if err != nil || len(videoURLs) == 0 {
				return nil, fmt.Errorf("jimeng returned no video URLs")
			}
			if err := downloadFile(ctx, videoURLs[0], req.OutputPath); err != nil {
				return nil, fmt.Errorf("jimeng download video: %w", err)
			}
			return &GenerateResponse{
				FilePath:   req.OutputPath,
				ModelUsed:  a.reqKey,
				Cost:       0.0,
				DurationMs: time.Since(start).Milliseconds(),
			}, nil
		}
	}
}

func (a *JimengVideoAdapter) HealthCheck(_ context.Context) error {
	return nil
}

// downloadFile fetches a URL and writes it to disk.
func downloadFile(ctx context.Context, url, outputPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

// extractString navigates a nested map[string]interface{} to extract a string value.
// Example: extractString(m, "data", "task_id") returns m["data"].(map[string]interface{})["task_id"].(string)
func extractString(m map[string]interface{}, keys ...string) (string, error) {
	current := interface{}(m)
	for i, key := range keys {
		cm, ok := current.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("expected map at key path %v, got %T", keys[:i], current)
		}
		current = cm[key]
	}
	s, ok := current.(string)
	if !ok {
		return "", fmt.Errorf("expected string at key path %v, got %T", keys, current)
	}
	return s, nil
}

// extractStringSlice navigates a nested map[string]interface{} to extract a []string value.
// The SDK returns []interface{} which we convert to []string.
func extractStringSlice(m map[string]interface{}, keys ...string) ([]string, error) {
	current := interface{}(m)
	for i, key := range keys {
		cm, ok := current.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("expected map at key path %v, got %T", keys[:i], current)
		}
		current = cm[key]
	}
	raw, ok := current.([]interface{})
	if !ok {
		return nil, fmt.Errorf("expected []interface{} at key path %v, got %T", keys, current)
	}
	result := make([]string, 0, len(raw))
	for _, v := range raw {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("expected string in slice, got %T", v)
		}
		result = append(result, s)
	}
	return result, nil
}
