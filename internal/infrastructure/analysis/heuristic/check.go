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

// okSignal reports that a check ran and found nothing suspicious.
// Type "meta" + weight 0 keeps it out of verdict scoring.
func okSignal(checkName, description string) domaincontent.Signal {
	return domaincontent.Signal{
		Type:        "meta",
		Code:        checkName + "_ok",
		Description: description,
		Weight:      0,
	}
}

// skippedSignal reports that a check could not apply (wrong format, missing deps, etc.).
func skippedSignal(checkName, description string) domaincontent.Signal {
	return domaincontent.Signal{
		Type:        "meta",
		Code:        checkName + "_skipped",
		Description: description,
		Weight:      0,
	}
}
