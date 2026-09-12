package presenter

import (
	"time"

	domainactivity "go-api/internal/domain/activity"
)

type ActivityItemResponse struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"`
	Message     string         `json:"message"`
	ActorType   string         `json:"actorType"`
	ActorName   string         `json:"actorName"`
	ActorUserID string         `json:"actorUserId,omitempty"`
	Payload     map[string]any `json:"payload,omitempty"`
	OccurredAt  time.Time      `json:"occurredAt"`
}

func NewActivityListResponseFromViews(views []domainactivity.View) []ActivityItemResponse {
	out := make([]ActivityItemResponse, 0, len(views))
	for _, view := range views {
		item := ActivityItemResponse{
			ID:         view.ID.String(),
			Type:       view.Type,
			Message:    view.Message,
			ActorType:  view.ActorType,
			ActorName:  view.ActorName,
			Payload:    view.Payload,
			OccurredAt: view.OccurredAt,
		}
		if view.ActorUserID != nil {
			item.ActorUserID = view.ActorUserID.String()
		}
		out = append(out, item)
	}
	return out
}
