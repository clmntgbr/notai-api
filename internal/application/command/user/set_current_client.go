package user

import (
	"context"
	"errors"

	domainclient "go-api/internal/domain/client"
	"go-api/internal/domain/port"
	domainuser "go-api/internal/domain/user"

	"github.com/google/uuid"
)

type SetCurrentClientCommand struct {
	UserID   uuid.UUID
	ClientID uuid.UUID
}

type SetCurrentClientHandler struct {
	userRepo   domainuser.UserWriteRepository
	clientRepo domainclient.ClientWriteRepository
	outbox     port.OutboxRepository
}

func NewSetCurrentClientHandler(
	userRepo domainuser.UserWriteRepository,
	clientRepo domainclient.ClientWriteRepository,
	outbox port.OutboxRepository,
) *SetCurrentClientHandler {
	return &SetCurrentClientHandler{
		userRepo:   userRepo,
		clientRepo: clientRepo,
		outbox:     outbox,
	}
}

func (h *SetCurrentClientHandler) Handle(ctx context.Context, cmd SetCurrentClientCommand) error {
	if cmd.UserID == uuid.Nil || cmd.ClientID == uuid.Nil {
		return errors.New("userId and clientId are required")
	}

	return h.userRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		client, err := h.clientRepo.GetByID(txCtx, cmd.ClientID)
		if err != nil {
			return errors.New("failed to get client")
		}
		if client == nil {
			return errors.New("client not found")
		}

		isMember := false
		for _, memberID := range client.MemberIDs {
			if memberID == cmd.UserID {
				isMember = true
				break
			}
		}
		if !isMember {
			// Same sentinel as missing client: do not reveal that the UUID exists.
			return errors.New("client not found")
		}

		user, err := h.userRepo.GetByID(txCtx, cmd.UserID)
		if err != nil {
			return errors.New("failed to get user")
		}
		if user == nil {
			return errors.New("user not found")
		}

		if !user.SetCurrentClient(cmd.ClientID) {
			return nil
		}

		if err := h.userRepo.Update(txCtx, user); err != nil {
			return errors.New("failed to set current client")
		}
		return h.outbox.StoreEvents(txCtx, user.PullEvents())
	})
}
