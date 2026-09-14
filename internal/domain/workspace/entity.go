package workspace

import (
	"time"

	"go-api/internal/domain/event"

	"github.com/google/uuid"
)

type Workspace struct {
	ID             uuid.UUID
	Name           string
	OwnerUserID    uuid.UUID
	SubscriptionID *uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time

	events []event.DomainEvent
}

func NewWorkspace(name string, ownerUserID uuid.UUID) *Workspace {
	now := time.Now().UTC()
	w := &Workspace{
		ID:          uuid.New(),
		Name:        name,
		OwnerUserID: ownerUserID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	w.recordEvent(WorkspaceCreated{
		ID:          uuid.New().String(),
		WorkspaceID: w.ID.String(),
		Name:        w.Name,
		OwnerUserID: w.OwnerUserID.String(),
		Timestamp:   now,
	})
	return w
}

func (w *Workspace) PullEvents() []event.DomainEvent {
	events := w.events
	w.events = nil
	return events
}

func (w *Workspace) recordEvent(e event.DomainEvent) {
	w.events = append(w.events, e)
}

func (w *Workspace) AssignSubscription(subscriptionID uuid.UUID) {
	id := subscriptionID
	w.SubscriptionID = &id
	w.UpdatedAt = time.Now().UTC()
	w.recordEvent(WorkspaceSubscriptionAssigned{
		ID:             uuid.New().String(),
		WorkspaceID:    w.ID.String(),
		SubscriptionID: subscriptionID.String(),
		Timestamp:      w.UpdatedAt,
	})
}
