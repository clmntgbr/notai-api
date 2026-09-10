package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	domaincampaign "go-api/internal/domain/campaign"
	domainclient "go-api/internal/domain/client"
	"go-api/internal/domain/port"
	domainuser "go-api/internal/domain/user"
)

type CreateUserCommand struct {
	ClerkID   string
	FirstName string
	LastName  string
	Email     string
	Banned    bool
}

type CreateUserHandler struct {
	userRepo     domainuser.UserWriteRepository
	clientRepo   domainclient.ClientWriteRepository
	campaignRepo domaincampaign.CampaignWriteRepository
	outbox       port.OutboxRepository
}

func NewCreateUserHandler(
	userRepo domainuser.UserWriteRepository,
	clientRepo domainclient.ClientWriteRepository,
	campaignRepo domaincampaign.CampaignWriteRepository,
	outbox port.OutboxRepository,
) *CreateUserHandler {
	return &CreateUserHandler{
		userRepo:     userRepo,
		clientRepo:   clientRepo,
		campaignRepo: campaignRepo,
		outbox:       outbox,
	}
}

func (h *CreateUserHandler) Handle(ctx context.Context, cmd CreateUserCommand) (*domainuser.User, error) {
	u := domainuser.NewUser(cmd.ClerkID, cmd.FirstName, cmd.LastName, cmd.Email, cmd.Banned)

	err := h.userRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.userRepo.Save(txCtx, u); err != nil {
			return err
		}

		client := domainclient.NewClient(personalClientName(cmd.FirstName, cmd.LastName), u.ID)
		client.AddMember(u.ID)
		if err := h.clientRepo.Save(txCtx, client); err != nil {
			return err
		}

		defaultCampaign := domaincampaign.NewDefaultCampaign(client.ID)
		if err := h.campaignRepo.Save(txCtx, defaultCampaign); err != nil {
			return err
		}

		u.SetCurrentClient(client.ID)
		if err := h.userRepo.Update(txCtx, u); err != nil {
			return err
		}

		events := append(u.PullEvents(), client.PullEvents()...)
		events = append(events, defaultCampaign.PullEvents()...)
		return h.outbox.StoreEvents(txCtx, events)
	})
	if err != nil {
		return nil, errors.New("failed to create user")
	}

	return u, nil
}

func personalClientName(firstName, lastName string) string {
	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)
	switch {
	case firstName != "" && lastName != "":
		return fmt.Sprintf("%s %s", firstName, lastName)
	case firstName != "":
		return fmt.Sprintf("%s's Client", firstName)
	default:
		return "Personal Client"
	}
}
