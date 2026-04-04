// internal/quality/checker_test.go
package quality

import (
	"context"
	"math"
	"testing"

	"github.com/west-garden/short-maker/internal/domain"
)

func TestDefaultDimensions_WeightsSumToOne(t *testing.T) {
	dims := DefaultDimensions(90)
	var total float64
	for _, d := range dims {
		total += d.Weight
	}
	if math.Abs(total-1.0) > 0.001 {
		t.Errorf("expected weights to sum to 1.0, got %f", total)
	}
}

func TestDefaultDimensions_Count(t *testing.T) {
	dims := DefaultDimensions(90)
	if len(dims) != 5 {
		t.Errorf("expected 5 dimensions, got %d", len(dims))
	}
}

func TestNewReport_WeightedScore(t *testing.T) {
	report := NewReport(1, DefaultDimensions(80), domain.GradeB)
	if report.TotalScore != 80 {
		t.Errorf("expected total score 80, got %d", report.TotalScore)
	}
	if !report.Passed {
		t.Error("expected Passed=true for score 80 with grade B")
	}
}

func TestNewReport_FailsBelowThreshold(t *testing.T) {
	report := NewReport(1, DefaultDimensions(60), domain.GradeS)
	if report.TotalScore != 60 {
		t.Errorf("expected total score 60, got %d", report.TotalScore)
	}
	if report.Passed {
		t.Error("expected Passed=false for score 60 with grade S")
	}
}

func TestMockChecker_AlwaysPasses(t *testing.T) {
	checker := NewMockChecker()
	shot := domain.NewShotSpec(1, 1)
	report, err := checker.Check(context.Background(), "/tmp/test.png", shot, nil)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !report.Passed {
		t.Error("expected MockChecker to always pass")
	}
	if report.TotalScore != 90 {
		t.Errorf("expected score 90, got %d", report.TotalScore)
	}
	if len(report.Dimensions) != 5 {
		t.Errorf("expected 5 dimensions, got %d", len(report.Dimensions))
	}
}
