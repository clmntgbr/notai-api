package client

import (
	"context"
	"errors"

	cmdquota "go-api/internal/application/command/quota"
	domainclient "go-api/internal/domain/client"
	"go-api/internal/domain/port"
	domainuser "go-api/internal/domain/user"

	"github.com/google/uuid"
)

type AddClientMemberCommand struct {
	ClientID    uuid.UUID
	UserID      uuid.UUID
	ActorUserID uuid.UUID
}

type AddClientMemberHandler struct {
	clientRepo domainclient.ClientWriteRepository
	userRepo   domainuser.UserWriteRepository
	quota      *cmdquota.AssertCreateAllowedHandler
	outbox     port.OutboxRepository
}

func NewAddClientMemberHandler(
	clientRepo domainclient.ClientWriteRepository,
	userRepo domainuser.UserWriteRepository,
	quota *cmdquota.AssertCreateAllowedHandler,
	outbox port.OutboxRepository,
) *AddClientMemberHandler {
	return &AddClientMemberHandler{
		clientRepo: clientRepo,
		userRepo:   userRepo,
		quota:      quota,
		outbox:     outbox,
	}
}

func (h *AddClientMemberHandler) Handle(ctx context.Context, cmd AddClientMemberCommand) error {
	if cmd.ClientID == uuid.Nil || cmd.UserID == uuid.Nil || cmd.ActorUserID == uuid.Nil {
		return errors.New("clientId, userId and actor are required")
	}

	return h.clientRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		client, err := h.clientRepo.GetByID(txCtx, cmd.ClientID)
		if err != nil {
			return errors.New("failed to get client")
		}
		if client == nil {
			return errors.New("client not found")
		}

		actorIsMember := false
		for _, id := range client.MemberIDs {
			if id == cmd.ActorUserID {
				actorIsMember = true
				break
			}
		}
		if !actorIsMember {
			return errors.New("client not found")
		}

		user, err := h.userRepo.GetByID(txCtx, cmd.UserID)
		if err != nil {
			return errors.New("failed to get user")
		}
		if user == nil {
			return errors.New("user not found")
		}

		alreadyMember := false
		for _, id := range client.MemberIDs {
			if id == cmd.UserID {
				alreadyMember = true
				break
			}
		}
		if alreadyMember {
			return nil
		}

		if h.quota != nil {
			if err := h.quota.AssertMemberCreate(txCtx, cmd.ClientID); err != nil {
				return err
			}
		}

		if !client.AddMember(cmd.UserID) {
			return nil
		}

		if err := h.clientRepo.Update(txCtx, client); err != nil {
			return errors.New("failed to add client member")
		}

		return h.outbox.StoreEvents(txCtx, client.PullEvents())
	})
}
