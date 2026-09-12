package heuristic

import (
	"context"

	domaincontent "go-api/internal/domain/content"
)

// GeometryCheck is disabled by default until a landmark/ONNX runtime is wired.
type GeometryCheck struct{}

func (GeometryCheck) Name() string { return "geometry" }

func (GeometryCheck) Run(ctx context.Context, in CheckInput, p CheckParams) ([]domaincontent.Signal, error) {
	_ = ctx
	_ = in
	_ = p
	return nil, nil
}
