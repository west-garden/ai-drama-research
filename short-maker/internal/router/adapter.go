package router

import "context"

// ModelType distinguishes image vs video generation models.
type ModelType string

const (
	ModelTypeImage ModelType = "image"
	ModelTypeVideo ModelType = "video"
)

// Capabilities declares what a model adapter can do.
type Capabilities struct {
	Type           ModelType
	Styles         []string // manga, 3d, live_action
	MaxResolution  string   // e.g. "1024x1024"
	SupportsFusion bool     // supports character reference injection
}

// GenerateRequest is the unified input for all generation adapters.
type GenerateRequest struct {
	Prompt        string            // generation prompt
	Style         string            // target style
	CharacterRefs []string          // character reference image paths (for fusion)
	CameraMove    string            // camera movement directive (video only)
	SourceImage   string            // source image path (video only)
	OutputPath    string            // where to write the output file
	Metadata      map[string]string // extension parameters
}

// GenerateResponse is the unified output from generation adapters.
type GenerateResponse struct {
	FilePath   string  // actual output file path
	ModelUsed  string  // which model was used
	Cost       float64 // cost in USD
	DurationMs int64   // wall-clock milliseconds
}

// ModelAdapter is the interface every generation model must implement.
// Adding a new model = implementing this interface.
type ModelAdapter interface {
	Name() string
	Capabilities() Capabilities
	Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
	HealthCheck(ctx context.Context) error
}
