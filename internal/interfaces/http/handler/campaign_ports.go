package handler

import (
	"context"
	"io"

	campaigncmd "go-api/internal/application/command/campaign"
	querycampaign "go-api/internal/application/query/campaign"
	queryclient "go-api/internal/application/query/client"
	domaincampaign "go-api/internal/domain/campaign"
	domainclient "go-api/internal/domain/client"
)

type campaignCreateHandler interface {
	Handle(ctx context.Context, cmd campaigncmd.CreateCampaignCommand) (*domaincampaign.Campaign, error)
}

type campaignUpdateHandler interface {
	Handle(ctx context.Context, cmd campaigncmd.UpdateCampaignCommand) error
}

type campaignDeleteHandler interface {
	Handle(ctx context.Context, cmd campaigncmd.DeleteCampaignCommand) error
}

type campaignGetByIDHandler interface {
	Handle(ctx context.Context, q querycampaign.GetCampaignByIDQuery) (*domaincampaign.CampaignView, error)
}

type campaignListByClientHandler interface {
	Handle(ctx context.Context, q querycampaign.ListCampaignsByClientQuery) ([]domaincampaign.CampaignView, int64, error)
}

type campaignGetClientByIDHandler interface {
	Handle(ctx context.Context, q queryclient.GetClientByIDQuery) (*domainclient.ClientView, error)
}

type campaignPresignBackgroundHandler interface {
	Handle(ctx context.Context, cmd campaigncmd.PresignBackgroundCommand) (*campaigncmd.PresignBackgroundResult, error)
}

type campaignClearBackgroundHandler interface {
	Handle(ctx context.Context, cmd campaigncmd.ClearBackgroundCommand) error
}

type campaignStorage interface {
	GetThumbnail(ctx context.Context, key string) (io.ReadCloser, error)
}
