// internal/quality/mock_checker.go
package quality

import (
	"context"

	"github.com/west-garden/short-maker/internal/domain"
)

type MockChecker struct{}

func NewMockChecker() *MockChecker {
	return &MockChecker{}
}

func (c *MockChecker) Check(_ context.Context, _ string, shotSpec *domain.ShotSpec, _ []*domain.Asset) (*QualityReport, error) {
	return NewReport(shotSpec.ShotNumber, DefaultDimensions(90), domain.GradeC), nil
}
