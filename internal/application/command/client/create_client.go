package client

import (
	"context"
	"errors"

	domaincampaign "go-api/internal/domain/campaign"
	domainclient "go-api/internal/domain/client"
	"go-api/internal/domain/port"
	domainuser "go-api/internal/domain/user"
	domainworkspace "go-api/internal/domain/workspace"

	"github.com/google/uuid"
)

type CreateClientCommand struct {
	Name          string
	CreatorUserID uuid.UUID
}

type CreateClientHandler struct {
	clientRepo    domainclient.ClientWriteRepository
	workspaceRepo domainworkspace.WorkspaceWriteRepository
	campaignRepo  domaincampaign.CampaignWriteRepository
	userRepo      domainuser.UserWriteRepository
	outbox        port.OutboxRepository
}

func NewCreateClientHandler(
	clientRepo domainclient.ClientWriteRepository,
	workspaceRepo domainworkspace.WorkspaceWriteRepository,
	campaignRepo domaincampaign.CampaignWriteRepository,
	userRepo domainuser.UserWriteRepository,
	outbox port.OutboxRepository,
) *CreateClientHandler {
	return &CreateClientHandler{
		clientRepo:    clientRepo,
		workspaceRepo: workspaceRepo,
		campaignRepo:  campaignRepo,
		userRepo:      userRepo,
		outbox:        outbox,
	}
}

func (h *CreateClientHandler) Handle(
	ctx context.Context,
	cmd CreateClientCommand,
) (*domainclient.Client, error) {
	if cmd.Name == "" {
		return nil, errors.New("name is required")
	}
	if cmd.CreatorUserID == uuid.Nil {
		return nil, errors.New("creator user is required")
	}

	var client *domainclient.Client

	err := h.clientRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		workspace, err := h.workspaceRepo.GetByOwnerUserID(txCtx, cmd.CreatorUserID)
		if err != nil {
			return errors.New("failed to get workspace")
		}
		if workspace == nil {
			return errors.New("workspace not found")
		}

		client = domainclient.NewClient(cmd.Name, workspace.ID, cmd.CreatorUserID)
		client.AddMember(cmd.CreatorUserID)

		if err := h.clientRepo.Save(txCtx, client); err != nil {
			return err
		}

		defaultCampaign := domaincampaign.NewDefaultCampaign(client.ID)
		if err := h.campaignRepo.Save(txCtx, defaultCampaign); err != nil {
			return err
		}

		user, err := h.userRepo.GetByID(txCtx, cmd.CreatorUserID)
		if err != nil {
			return errors.New("failed to get creator user")
		}
		if user == nil {
			return errors.New("creator user not found")
		}

		user.SetCurrentClient(client.ID)
		if err := h.userRepo.Update(txCtx, user); err != nil {
			return errors.New("failed to set current client")
		}

		events := append(client.PullEvents(), defaultCampaign.PullEvents()...)
		events = append(events, user.PullEvents()...)
		return h.outbox.StoreEvents(txCtx, events)
	})
	if err != nil {
		if err.Error() == "creator user not found" || err.Error() == "failed to get creator user" ||
			err.Error() == "failed to set current client" || err.Error() == "workspace not found" ||
			err.Error() == "failed to get workspace" {
			return nil, err
		}
		return nil, errors.New("failed to create client")
	}

	return client, nil
}
