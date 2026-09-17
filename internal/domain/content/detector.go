package content

import (
	"context"

	"github.com/google/uuid"
)

// Detector runs one analysis source against a content item.
type Detector interface {
	Name() string
	ExpectedWeight() float64
	Analyze(ctx context.Context, contentID uuid.UUID, objectKey string) ([]Signal, error)
}

// ExternalDetector is optionally implemented by paid/third-party detectors.
// Local heuristics must not implement this (or must return false).
type ExternalDetector interface {
	IsExternal() bool
}

// IsExternal reports whether d is a third-party detector subject to plan caps.
func IsExternal(d Detector) bool {
	if e, ok := d.(ExternalDetector); ok {
		return e.IsExternal()
	}
	return false
}

// RulesetVersionTracker is optionally implemented by detectors that persist
// a calibration/ruleset version used during Analyze.
type RulesetVersionTracker interface {
	TakeRulesetVersion(contentID uuid.UUID) *int
}
