// internal/router/router.go
package router

import (
	"context"
	"fmt"

	"github.com/west-garden/short-maker/internal/domain"
)

// ModelRouter selects the best adapter for a given grade+style+type combination.
type ModelRouter struct {
	adapters []ModelAdapter
}

// NewModelRouter creates a router with the given adapters.
func NewModelRouter(adapters ...ModelAdapter) *ModelRouter {
	return &ModelRouter{adapters: adapters}
}

// Route finds the first adapter matching the requested model type and style.
func (r *ModelRouter) Route(grade domain.Grade, style string, modelType ModelType) (ModelAdapter, error) {
	for _, a := range r.adapters {
		caps := a.Capabilities()
		if caps.Type != modelType {
			continue
		}
		if !hasStyle(caps.Styles, style) {
			continue
		}
		return a, nil
	}
	return nil, fmt.Errorf("no adapter found for type=%s style=%s grade=%s", modelType, style, grade)
}

// Generate routes and then generates in one call.
func (r *ModelRouter) Generate(ctx context.Context, grade domain.Grade, style string, modelType ModelType, req GenerateRequest) (*GenerateResponse, error) {
	adapter, err := r.Route(grade, style, modelType)
	if err != nil {
		return nil, err
	}
	return adapter.Generate(ctx, req)
}

func hasStyle(styles []string, target string) bool {
	for _, s := range styles {
		if s == target {
			return true
		}
	}
	return false
}
