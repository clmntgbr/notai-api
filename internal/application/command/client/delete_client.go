package client

import (
	"context"
	"errors"

	domainclient "go-api/internal/domain/client"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type DeleteClientHandler struct {
	repo   domainclient.ClientWriteRepository
	outbox port.OutboxRepository
}

func NewDeleteClientHandler(
	repo domainclient.ClientWriteRepository,
	outbox port.OutboxRepository,
) *DeleteClientHandler {
	return &DeleteClientHandler{repo: repo, outbox: outbox}
}

func (h *DeleteClientHandler) Handle(ctx context.Context, id uuid.UUID) error {
	return h.repo.WithTransaction(ctx, func(txCtx context.Context) error {
		client, err := h.repo.GetByID(txCtx, id)
		if err != nil {
			return errors.New("failed to get client")
		}
		if client == nil {
			return nil
		}

		client.MarkDeleted()
		events := client.PullEvents()

		if err := h.repo.Delete(txCtx, client.ID); err != nil {
			return errors.New("failed to delete client")
		}

		return h.outbox.StoreEvents(txCtx, events)
	})
}
