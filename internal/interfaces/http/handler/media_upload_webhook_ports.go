package handler

import (
	"context"

	campaigncmd "go-api/internal/application/command/campaign"
)

type mediaUploadProcessHandler interface {
	Handle(ctx context.Context, cmd campaigncmd.ProcessBackgroundUploadCommand) error
}
