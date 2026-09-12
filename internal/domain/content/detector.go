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
