package metadata

import (
	"context"
	"fmt"

	domaincontent "go-api/internal/domain/content"

	"github.com/google/uuid"
)

const DetectorName = "metadata"

// Detector extracts lightweight metadata signals from the content record.
type Detector struct{}

func New() *Detector { return &Detector{} }

func (d *Detector) Name() string { return DetectorName }

func (d *Detector) ExpectedWeight() float64 { return 1 }

func (d *Detector) Analyze(
	ctx context.Context,
	contentID uuid.UUID,
	objectKey string,
) ([]domaincontent.Signal, error) {
	_ = ctx
	_ = contentID
	return []domaincontent.Signal{
		{
			Type:        "meta",
			Code:        "object_present",
			Description: fmt.Sprintf("Object key recorded: %s", objectKey),
			Weight:      0.1,
		},
	}, nil
}
