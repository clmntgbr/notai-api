package workspace

import (
	"time"

	"go-api/internal/domain/event"
)

const (
	EventTypeWorkspaceCreated              = "workspace.created.v1"
	EventTypeWorkspaceSubscriptionAssigned = "workspace.subscription_assigned.v1"
)

type WorkspaceCreated struct {
	ID          string    `json:"eventId"`
	WorkspaceID string    `json:"workspaceId"`
	Name        string    `json:"name"`
	OwnerUserID string    `json:"ownerUserId"`
	Timestamp   time.Time `json:"timestamp"`
}

func (e WorkspaceCreated) EventID() string       { return e.ID }
func (e WorkspaceCreated) EventType() string     { return EventTypeWorkspaceCreated }
func (e WorkspaceCreated) AggregateID() string   { return e.WorkspaceID }
func (e WorkspaceCreated) OccurredAt() time.Time { return e.Timestamp }

type WorkspaceSubscriptionAssigned struct {
	ID             string    `json:"eventId"`
	WorkspaceID    string    `json:"workspaceId"`
	SubscriptionID string    `json:"subscriptionId"`
	Timestamp      time.Time `json:"timestamp"`
}

func (e WorkspaceSubscriptionAssigned) EventID() string { return e.ID }
func (e WorkspaceSubscriptionAssigned) EventType() string {
	return EventTypeWorkspaceSubscriptionAssigned
}
func (e WorkspaceSubscriptionAssigned) AggregateID() string   { return e.WorkspaceID }
func (e WorkspaceSubscriptionAssigned) OccurredAt() time.Time { return e.Timestamp }

var (
	_ event.DomainEvent = WorkspaceCreated{}
	_ event.DomainEvent = WorkspaceSubscriptionAssigned{}
)
