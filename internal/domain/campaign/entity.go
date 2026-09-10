package campaign

import (
	"time"

	"go-api/internal/domain/event"

	"github.com/google/uuid"
)

type Campaign struct {
	ID        uuid.UUID
	ClientID  uuid.UUID
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time

	events []event.DomainEvent
}

func NewCampaign(name string, clientID uuid.UUID) *Campaign {
	now := time.Now().UTC()
	c := &Campaign{
		ID:        uuid.New(),
		ClientID:  clientID,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
	c.recordEvent(CampaignCreated{
		ID:         uuid.New().String(),
		CampaignID: c.ID.String(),
		ClientID:   clientID.String(),
		Name:       c.Name,
		Timestamp:  now,
	})
	return c
}

func (c *Campaign) PullEvents() []event.DomainEvent {
	events := c.events
	c.events = nil
	return events
}

func (c *Campaign) recordEvent(e event.DomainEvent) {
	c.events = append(c.events, e)
}

func (c *Campaign) ApplyUpdate(name string) {
	c.Name = name
	c.UpdatedAt = time.Now().UTC()
	c.recordEvent(CampaignUpdated{
		ID:         uuid.New().String(),
		CampaignID: c.ID.String(),
		ClientID:   c.ClientID.String(),
		Name:       c.Name,
		Timestamp:  c.UpdatedAt,
	})
}

func (c *Campaign) MarkDeleted() {
	c.recordEvent(CampaignDeleted{
		ID:         uuid.New().String(),
		CampaignID: c.ID.String(),
		ClientID:   c.ClientID.String(),
		Timestamp:  time.Now().UTC(),
	})
}
