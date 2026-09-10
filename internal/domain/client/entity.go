package client

import (
	"time"

	"go-api/internal/domain/event"

	"github.com/google/uuid"
)

type Client struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
	MemberIDs []uuid.UUID

	events []event.DomainEvent
}

func NewClient(name string, createdByUserID uuid.UUID) *Client {
	now := time.Now().UTC()
	c := &Client{
		ID:        uuid.New(),
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
		MemberIDs: nil,
	}
	c.recordEvent(ClientCreated{
		ID:              uuid.New().String(),
		ClientID:        c.ID.String(),
		Name:            c.Name,
		CreatedByUserID: createdByUserID.String(),
		Timestamp:       now,
	})
	return c
}

func (c *Client) PullEvents() []event.DomainEvent {
	events := c.events
	c.events = nil
	return events
}

func (c *Client) recordEvent(e event.DomainEvent) {
	c.events = append(c.events, e)
}

func (c *Client) ApplyUpdate(name string) {
	c.Name = name
	c.UpdatedAt = time.Now().UTC()
	c.recordEvent(ClientUpdated{
		ID:        uuid.New().String(),
		ClientID:  c.ID.String(),
		Name:      c.Name,
		MemberIDs: uuidStrings(c.MemberIDs),
		Timestamp: c.UpdatedAt,
	})
}

func (c *Client) MarkDeleted() {
	c.recordEvent(ClientDeleted{
		ID:        uuid.New().String(),
		ClientID:  c.ID.String(),
		MemberIDs: uuidStrings(c.MemberIDs),
		Timestamp: time.Now().UTC(),
	})
}

func uuidStrings(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}

func (c *Client) AddMember(userID uuid.UUID) bool {
	for _, id := range c.MemberIDs {
		if id == userID {
			return false
		}
	}
	c.MemberIDs = append(c.MemberIDs, userID)
	c.UpdatedAt = time.Now().UTC()
	c.recordEvent(ClientMemberAdded{
		ID:        uuid.New().String(),
		ClientID:  c.ID.String(),
		UserID:    userID.String(),
		Timestamp: c.UpdatedAt,
	})
	return true
}

func (c *Client) RemoveMember(userID uuid.UUID) bool {
	idx := -1
	for i, id := range c.MemberIDs {
		if id == userID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false
	}
	c.MemberIDs = append(c.MemberIDs[:idx], c.MemberIDs[idx+1:]...)
	c.UpdatedAt = time.Now().UTC()
	c.recordEvent(ClientMemberRemoved{
		ID:        uuid.New().String(),
		ClientID:  c.ID.String(),
		UserID:    userID.String(),
		Timestamp: c.UpdatedAt,
	})
	return true
}
