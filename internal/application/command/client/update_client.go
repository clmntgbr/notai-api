package client

import (
	"context"
	"errors"

	domainclient "go-api/internal/domain/client"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type UpdateClientCommand struct {
	ID   uuid.UUID
	Name string
}

type UpdateClientHandler struct {
	repo   domainclient.ClientWriteRepository
	outbox port.OutboxRepository
}

func NewUpdateClientHandler(
	repo domainclient.ClientWriteRepository,
	outbox port.OutboxRepository,
) *UpdateClientHandler {
	return &UpdateClientHandler{repo: repo, outbox: outbox}
}

func (h *UpdateClientHandler) Handle(ctx context.Context, cmd UpdateClientCommand) error {
	if cmd.Name == "" {
		return errors.New("name is required")
	}

	return h.repo.WithTransaction(ctx, func(txCtx context.Context) error {
		client, err := h.repo.GetByID(txCtx, cmd.ID)
		if err != nil {
			return errors.New("failed to get client")
		}
		if client == nil {
			return errors.New("client not found")
		}

		client.ApplyUpdate(cmd.Name)

		if err := h.repo.Update(txCtx, client); err != nil {
			return errors.New("failed to update client")
		}
		return h.outbox.StoreEvents(txCtx, client.PullEvents())
	})
}
