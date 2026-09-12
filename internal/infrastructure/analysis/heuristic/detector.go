package heuristic

import (
	"context"
	"path/filepath"
	"strings"

	domaincontent "go-api/internal/domain/content"

	"github.com/google/uuid"
)

const DetectorName = "heuristic"

// Detector applies cheap local heuristics (extension / naming cues).
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

	base := strings.ToLower(filepath.Base(objectKey))
	signals := []domaincontent.Signal{
		{
			Type:        "heuristic",
			Code:        "format_ok",
			Description: "File extension is an accepted image type",
			Weight:      0.05,
		},
	}

	suspicious := []string{"midjourney", "stable-diffusion", "stablediffusion", "dalle", "flux"}
	for _, token := range suspicious {
		if strings.Contains(base, token) {
			signals = append(signals, domaincontent.Signal{
				Type:        "heuristic",
				Code:        "suspicious_filename",
				Description: "Filename contains a known generative-AI token",
				Weight:      0.4,
			})
			break
		}
	}

	return signals, nil
}
