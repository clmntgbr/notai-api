package client

import (
	"context"
	"errors"

	domainclient "go-api/internal/domain/client"
	"go-api/internal/domain/port"
	domainuser "go-api/internal/domain/user"

	"github.com/google/uuid"
)

type CreateClientCommand struct {
	Name          string
	CreatorUserID uuid.UUID
}

type CreateClientHandler struct {
	clientRepo domainclient.ClientWriteRepository
	userRepo   domainuser.UserWriteRepository
	outbox     port.OutboxRepository
}

func NewCreateClientHandler(
	clientRepo domainclient.ClientWriteRepository,
	userRepo domainuser.UserWriteRepository,
	outbox port.OutboxRepository,
) *CreateClientHandler {
	return &CreateClientHandler{
		clientRepo: clientRepo,
		userRepo:   userRepo,
		outbox:     outbox,
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

	client := domainclient.NewClient(cmd.Name, cmd.CreatorUserID)
	client.AddMember(cmd.CreatorUserID)

	err := h.clientRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.clientRepo.Save(txCtx, client); err != nil {
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

		events := append(client.PullEvents(), user.PullEvents()...)
		return h.outbox.StoreEvents(txCtx, events)
	})
	if err != nil {
		if err.Error() == "creator user not found" || err.Error() == "failed to get creator user" ||
			err.Error() == "failed to set current client" {
			return nil, err
		}
		return nil, errors.New("failed to create client")
	}

	return client, nil
}
