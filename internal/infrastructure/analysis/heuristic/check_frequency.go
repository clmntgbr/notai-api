package heuristic

import (
	"context"
	"fmt"

	domaincontent "go-api/internal/domain/content"
)

type FrequencyCheck struct{}

func (FrequencyCheck) Name() string { return "frequency" }

func (FrequencyCheck) Run(ctx context.Context, in CheckInput, p CheckParams) ([]domaincontent.Signal, error) {
	_ = ctx
	if in.Img == nil {
		return nil, nil
	}
	if err := ensureMinSize(in.Img, 32); err != nil {
		return nil, err
	}

	const size = 32
	gray := toGrayMatrix(in.Img, size)
	peakMin := p.Threshold("frequency_peak_min")
	if peakMin <= 0 {
		peakMin = 0.35
	}
	score := dftPeriodicityScore(gray, peakMin)
	maxScore := p.Threshold("frequency_periodicity_max")
	if maxScore <= 0 {
		maxScore = 0.55
	}
	if score > maxScore {
		return []domaincontent.Signal{{
			Type: "heuristic",
			Code: "frequency_artifact",
			Description: fmt.Sprintf(
				"Periodic pattern detected in frequency spectrum (score %.2f)",
				score,
			),
			Weight: p.WeightFor("frequency_artifact"),
		}}, nil
	}
	return nil, nil
}
