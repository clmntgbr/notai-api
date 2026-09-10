package client

import (
	"context"
	"errors"

	domainclient "go-api/internal/domain/client"
	"go-api/internal/domain/port"
	domainuser "go-api/internal/domain/user"

	"github.com/google/uuid"
)

type RemoveClientMemberCommand struct {
	ClientID uuid.UUID
	UserID   uuid.UUID
}

type RemoveClientMemberHandler struct {
	clientRepo domainclient.ClientWriteRepository
	userRepo   domainuser.UserWriteRepository
	outbox     port.OutboxRepository
}

func NewRemoveClientMemberHandler(
	clientRepo domainclient.ClientWriteRepository,
	userRepo domainuser.UserWriteRepository,
	outbox port.OutboxRepository,
) *RemoveClientMemberHandler {
	return &RemoveClientMemberHandler{
		clientRepo: clientRepo,
		userRepo:   userRepo,
		outbox:     outbox,
	}
}

func (h *RemoveClientMemberHandler) Handle(ctx context.Context, cmd RemoveClientMemberCommand) error {
	return h.clientRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		client, err := h.clientRepo.GetByID(txCtx, cmd.ClientID)
		if err != nil {
			return errors.New("failed to get client")
		}
		if client == nil {
			return errors.New("client not found")
		}

		if !client.RemoveMember(cmd.UserID) {
			return nil
		}

		if err := h.clientRepo.Update(txCtx, client); err != nil {
			return errors.New("failed to remove client member")
		}

		events := client.PullEvents()

		user, err := h.userRepo.GetByID(txCtx, cmd.UserID)
		if err != nil {
			return errors.New("failed to get user")
		}
		if user != nil &&
			user.CurrentClientID != nil &&
			*user.CurrentClientID == cmd.ClientID {
			user.ClearCurrentClient()
			if err := h.userRepo.Update(txCtx, user); err != nil {
				return errors.New("failed to clear current client")
			}
			events = append(events, user.PullEvents()...)
		}

		return h.outbox.StoreEvents(txCtx, events)
	})
}
