package activity

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go-api/internal/application/messaging"
	"go-api/internal/application/realtime"
	domainactivity "go-api/internal/domain/activity"
	domaincampaign "go-api/internal/domain/campaign"
	domainclient "go-api/internal/domain/client"
	domaincontent "go-api/internal/domain/content"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"
	domainuser "go-api/internal/domain/user"

	"github.com/google/uuid"
)

// Projector writes denormalized activity-feed rows from selected domain events.
type Projector struct {
	activityRepo      domainactivity.WriteRepository
	mediaRepo         domainmedia.MediaReadRepository
	userRepo          domainuser.UserReadRepository
	clientRepo        domainclient.ClientReadRepository
	realtimePublisher *realtime.Publisher
}

func NewProjector(
	activityRepo domainactivity.WriteRepository,
	mediaRepo domainmedia.MediaReadRepository,
	userRepo domainuser.UserReadRepository,
	clientRepo domainclient.ClientReadRepository,
	realtimePublisher port.RealtimePublisher,
) *Projector {
	return &Projector{
		activityRepo:      activityRepo,
		mediaRepo:         mediaRepo,
		userRepo:          userRepo,
		clientRepo:        clientRepo,
		realtimePublisher: realtime.NewPublisher(realtimePublisher),
	}
}

type realtimePayload struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"`
	Message     string         `json:"message"`
	ActorType   string         `json:"actorType"`
	ActorName   string         `json:"actorName"`
	ActorUserID string         `json:"actorUserId,omitempty"`
	Payload     map[string]any `json:"payload,omitempty"`
	OccurredAt  time.Time      `json:"occurredAt"`
	ClientID    string         `json:"clientId"`
}

func (p *Projector) OnContentVerdictRendered(ctx context.Context, payload []byte) error {
	// Media-level activity is preferred; per-content verdicts are not projected.
	_ = payload
	return nil
}

func (p *Projector) OnContentStatusChanged(ctx context.Context, payload []byte) error {
	// Media-level activity is preferred; per-content failures are not projected.
	_ = payload
	return nil
}

func (p *Projector) OnMediaVerdictRendered(ctx context.Context, payload []byte) error {
	var evt domainmedia.MediaVerdictRendered
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}

	clientID, err := uuid.Parse(evt.ClientID)
	if err != nil {
		return messaging.NonRetryable(err)
	}
	eventID, err := uuid.Parse(evt.ID)
	if err != nil {
		return messaging.NonRetryable(err)
	}
	mediaID, err := uuid.Parse(evt.MediaID)
	if err != nil {
		return messaging.NonRetryable(err)
	}

	filename := evt.MediaID
	if view, err := p.mediaRepo.FindByID(ctx, mediaID); err != nil {
		return messaging.Retryable(err)
	} else if view != nil && view.Filename != "" {
		filename = view.Filename
	}

	var activityType, message string
	switch domaincontent.Label(evt.Label) {
	case domaincontent.LabelAIGenerated:
		activityType = domainactivity.TypeMediaAIFlagged
		message = fmt.Sprintf(
			"“%s” flagged as AI-generated (%d/%d frames)",
			filename,
			evt.FlaggedCount,
			evt.TotalCount,
		)
	case domaincontent.LabelUncertain:
		activityType = domainactivity.TypeMediaManualReview
		message = fmt.Sprintf(
			"“%s” placed under manual review (%d/%d units)",
			filename,
			evt.FlaggedCount,
			evt.TotalCount,
		)
	case domaincontent.LabelHuman:
		activityType = domainactivity.TypeMediaHumanVerified
		message = fmt.Sprintf("“%s” verified as human content", filename)
	default:
		return nil
	}

	return p.insert(ctx, &domainactivity.Event{
		ID:        eventID,
		ClientID:  clientID,
		Type:      activityType,
		ActorType: domainactivity.ActorTypeSystem,
		ActorName: domainactivity.ActorNameSystem,
		Message:   message,
		Payload: map[string]any{
			"mediaId":      evt.MediaID,
			"campaignId":   evt.CampaignID,
			"filename":     filename,
			"label":        evt.Label,
			"flaggedCount": evt.FlaggedCount,
			"totalCount":   evt.TotalCount,
			"failedCount":  evt.FailedCount,
		},
		OccurredAt: evt.Timestamp,
	})
}

func (p *Projector) OnMediaStatusChanged(ctx context.Context, payload []byte) error {
	var evt domainmedia.MediaStatusChanged
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	if domainmedia.Status(evt.Status) != domainmedia.StatusFailed {
		return nil
	}

	clientID, err := uuid.Parse(evt.ClientID)
	if err != nil {
		return messaging.NonRetryable(err)
	}
	eventID, err := uuid.Parse(evt.ID)
	if err != nil {
		return messaging.NonRetryable(err)
	}
	mediaID, err := uuid.Parse(evt.MediaID)
	if err != nil {
		return messaging.NonRetryable(err)
	}

	filename := evt.MediaID
	if view, err := p.mediaRepo.FindByID(ctx, mediaID); err != nil {
		return messaging.Retryable(err)
	} else if view != nil && view.Filename != "" {
		filename = view.Filename
	}

	return p.insert(ctx, &domainactivity.Event{
		ID:        eventID,
		ClientID:  clientID,
		Type:      domainactivity.TypeMediaFailed,
		ActorType: domainactivity.ActorTypeSystem,
		ActorName: domainactivity.ActorNameSystem,
		Message:   fmt.Sprintf("“%s” failed during processing", filename),
		Payload: map[string]any{
			"mediaId":    evt.MediaID,
			"campaignId": evt.CampaignID,
			"filename":   filename,
			"status":     evt.Status,
		},
		OccurredAt: evt.Timestamp,
	})
}

func (p *Projector) OnCampaignCreated(ctx context.Context, payload []byte) error {
	var evt domaincampaign.CampaignCreated
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	if evt.IsDefault {
		return nil
	}

	clientID, err := uuid.Parse(evt.ClientID)
	if err != nil {
		return messaging.NonRetryable(err)
	}
	eventID, err := uuid.Parse(evt.ID)
	if err != nil {
		return messaging.NonRetryable(err)
	}

	clientName := "the client"
	if view, err := p.clientRepo.FindByID(ctx, clientID); err != nil {
		return messaging.Retryable(err)
	} else if view != nil && view.Name != "" {
		clientName = view.Name
	}

	message := fmt.Sprintf(
		"New campaign “%s” created for %s",
		evt.Name,
		clientName,
	)

	return p.insert(ctx, &domainactivity.Event{
		ID:        eventID,
		ClientID:  clientID,
		Type:      domainactivity.TypeCampaignCreated,
		ActorType: domainactivity.ActorTypeSystem,
		ActorName: domainactivity.ActorNameSystem,
		Message:   message,
		Payload: map[string]any{
			"campaignId": evt.CampaignID,
			"name":       evt.Name,
			"clientName": clientName,
		},
		OccurredAt: evt.Timestamp,
	})
}

func (p *Projector) OnClientMemberAdded(ctx context.Context, payload []byte) error {
	var evt domainclient.ClientMemberAdded
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}

	clientID, err := uuid.Parse(evt.ClientID)
	if err != nil {
		return messaging.NonRetryable(err)
	}
	eventID, err := uuid.Parse(evt.ID)
	if err != nil {
		return messaging.NonRetryable(err)
	}
	userID, err := uuid.Parse(evt.UserID)
	if err != nil {
		return messaging.NonRetryable(err)
	}

	memberName := "A member"
	if view, err := p.userRepo.FindByID(ctx, userID); err != nil {
		return messaging.Retryable(err)
	} else if view != nil {
		memberName = displayName(view.FirstName, view.LastName, view.Email)
	}

	message := fmt.Sprintf("%s joined the team", memberName)

	return p.insert(ctx, &domainactivity.Event{
		ID:        eventID,
		ClientID:  clientID,
		Type:      domainactivity.TypeClientMemberAdded,
		ActorType: domainactivity.ActorTypeSystem,
		ActorName: domainactivity.ActorNameSystem,
		Message:   message,
		Payload: map[string]any{
			"userId":     evt.UserID,
			"memberName": memberName,
		},
		OccurredAt: evt.Timestamp,
	})
}

func (p *Projector) insert(ctx context.Context, event *domainactivity.Event) error {
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	inserted, err := p.activityRepo.Insert(ctx, event)
	if err != nil {
		return messaging.Retryable(err)
	}
	if !inserted {
		return nil
	}
	return p.publishCreated(ctx, event)
}

func (p *Projector) publishCreated(ctx context.Context, event *domainactivity.Event) error {
	view, err := p.clientRepo.FindByID(ctx, event.ClientID)
	if err != nil {
		return messaging.Retryable(err)
	}
	if view == nil || len(view.MemberIDs) == 0 {
		return nil
	}

	payload := realtimePayload{
		ID:         event.ID.String(),
		Type:       event.Type,
		Message:    event.Message,
		ActorType:  event.ActorType,
		ActorName:  event.ActorName,
		Payload:    event.Payload,
		OccurredAt: event.OccurredAt,
		ClientID:   event.ClientID.String(),
	}
	if event.ActorUserID != nil {
		payload.ActorUserID = event.ActorUserID.String()
	}

	return p.realtimePublisher.ToMembers(
		ctx,
		realtime.EntityActivity,
		realtime.ActionCreated,
		view.MemberIDs,
		payload,
	)
}

func displayName(firstName, lastName, email string) string {
	name := strings.TrimSpace(strings.TrimSpace(firstName) + " " + strings.TrimSpace(lastName))
	if name != "" {
		return name
	}
	email = strings.TrimSpace(email)
	if email != "" {
		return email
	}
	return "A member"
}
