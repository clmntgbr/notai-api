package heuristic

import (
	"context"

	domaincontent "go-api/internal/domain/content"
)

// PRNUCheck is disabled by default until an external PRNU runtime is wired.
type PRNUCheck struct{}

func (PRNUCheck) Name() string { return "prnu" }

func (PRNUCheck) Run(ctx context.Context, in CheckInput, p CheckParams) ([]domaincontent.Signal, error) {
	_ = ctx
	_ = in
	_ = p
	return nil, nil
}
