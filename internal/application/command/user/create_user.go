package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	domaincampaign "go-api/internal/domain/campaign"
	domainclient "go-api/internal/domain/client"
	domainplan "go-api/internal/domain/plan"
	"go-api/internal/domain/port"
	domainsubscription "go-api/internal/domain/subscription"
	domainuser "go-api/internal/domain/user"
	domainworkspace "go-api/internal/domain/workspace"
)

type CreateUserCommand struct {
	ClerkID   string
	FirstName string
	LastName  string
	Email     string
	Banned    bool
}

type CreateUserHandler struct {
	userRepo         domainuser.UserWriteRepository
	workspaceRepo    domainworkspace.WorkspaceWriteRepository
	clientRepo       domainclient.ClientWriteRepository
	campaignRepo     domaincampaign.CampaignWriteRepository
	planRepo         domainplan.PlanWriteRepository
	subscriptionRepo domainsubscription.SubscriptionWriteRepository
	outbox           port.OutboxRepository
}

func NewCreateUserHandler(
	userRepo domainuser.UserWriteRepository,
	workspaceRepo domainworkspace.WorkspaceWriteRepository,
	clientRepo domainclient.ClientWriteRepository,
	campaignRepo domaincampaign.CampaignWriteRepository,
	planRepo domainplan.PlanWriteRepository,
	subscriptionRepo domainsubscription.SubscriptionWriteRepository,
	outbox port.OutboxRepository,
) *CreateUserHandler {
	return &CreateUserHandler{
		userRepo:         userRepo,
		workspaceRepo:    workspaceRepo,
		clientRepo:       clientRepo,
		campaignRepo:     campaignRepo,
		planRepo:         planRepo,
		subscriptionRepo: subscriptionRepo,
		outbox:           outbox,
	}
}

func (h *CreateUserHandler) Handle(ctx context.Context, cmd CreateUserCommand) (*domainuser.User, error) {
	u := domainuser.NewUser(cmd.ClerkID, cmd.FirstName, cmd.LastName, cmd.Email, cmd.Banned)

	err := h.userRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.userRepo.Save(txCtx, u); err != nil {
			return err
		}

		workspaceName := personalWorkspaceName(cmd.FirstName, cmd.LastName)
		workspace := domainworkspace.NewWorkspace(workspaceName, u.ID)

		freePlan, err := h.planRepo.GetBySlug(txCtx, domainplan.FreePlanSlug)
		if err != nil || freePlan == nil {
			return errors.New("free plan not found")
		}
		sub := domainsubscription.NewFreeSubscription(freePlan.ID)
		if err := h.subscriptionRepo.Save(txCtx, sub); err != nil {
			return errors.New("failed to create free subscription")
		}
		workspace.AssignSubscription(sub.ID)

		if err := h.workspaceRepo.Save(txCtx, workspace); err != nil {
			return err
		}

		client := domainclient.NewClient(personalClientName(cmd.FirstName, cmd.LastName), workspace.ID, u.ID)
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

		events := append(u.PullEvents(), workspace.PullEvents()...)
		events = append(events, client.PullEvents()...)
		events = append(events, defaultCampaign.PullEvents()...)
		events = append(events, sub.PullEvents()...)
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

func personalWorkspaceName(firstName, lastName string) string {
	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)
	switch {
	case firstName != "" && lastName != "":
		return fmt.Sprintf("%s %s", firstName, lastName)
	case firstName != "":
		return fmt.Sprintf("%s's Workspace", firstName)
	default:
		return "Personal Workspace"
	}
}
