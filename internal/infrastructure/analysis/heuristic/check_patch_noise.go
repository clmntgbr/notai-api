package heuristic

import (
	"context"
	"fmt"

	domaincontent "go-api/internal/domain/content"
)

type PatchNoiseCheck struct{}

func (PatchNoiseCheck) Name() string { return "patch_noise" }

func (PatchNoiseCheck) Run(ctx context.Context, in CheckInput, p CheckParams) ([]domaincontent.Signal, error) {
	_ = ctx
	if in.Img == nil {
		return nil, nil
	}
	if err := ensureMinSize(in.Img, 64); err != nil {
		return nil, err
	}

	grid := p.IntParam("patch_grid_size")
	if grid <= 0 {
		grid = 8
	}
	bounds := in.Img.Bounds()
	pw := bounds.Dx() / grid
	ph := bounds.Dy() / grid
	if pw < 4 || ph < 4 {
		return nil, nil
	}

	variances := make([]float64, 0, grid*grid)
	for gy := 0; gy < grid; gy++ {
		for gx := 0; gx < grid; gx++ {
			x0 := bounds.Min.X + gx*pw
			y0 := bounds.Min.Y + gy*ph
			x1 := x0 + pw
			y1 := y0 + ph
			variances = append(variances, laplacianVariance(in.Img, x0, y0, x1, y1))
		}
	}

	mean, std := meanStd(variances)
	threshold := p.Threshold("patch_variance_stddev")
	if threshold <= 0 {
		threshold = 2.0
	}
	outliers := 0
	for _, v := range variances {
		if std == 0 {
			continue
		}
		if (v-mean)/std > threshold || (mean-v)/std > threshold {
			outliers++
		}
	}

	maxOutliers := p.IntParam("patch_outlier_max_count")
	if maxOutliers <= 0 {
		maxOutliers = 6
	}
	if outliers > maxOutliers {
		return []domaincontent.Signal{{
			Type: "heuristic",
			Code: "patch_noise_inconsistent",
			Description: fmt.Sprintf(
				"%d regions with inconsistent sharpness relative to the rest of the image",
				outliers,
			),
			Weight: p.WeightFor("patch_noise_inconsistent"),
		}}, nil
	}
	return nil, nil
}
