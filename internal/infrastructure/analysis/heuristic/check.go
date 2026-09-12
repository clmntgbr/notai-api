package heuristic

import (
	"context"
	"image"

	domaincontent "go-api/internal/domain/content"

	"github.com/google/uuid"
)

// CheckInput is shared decoded state for all internal heuristic checks.
type CheckInput struct {
	ContentID uuid.UUID
	ObjectKey string
	RawBytes  []byte
	Img       image.Image
	Format    string
}

// Check is one internal heuristic analysis step.
type Check interface {
	Name() string
	Run(ctx context.Context, in CheckInput, params CheckParams) ([]domaincontent.Signal, error)
}
