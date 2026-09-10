package campaign

import (
	"errors"
	"time"

	"go-api/internal/domain/event"

	"github.com/google/uuid"
)

var (
	ErrDefaultCampaignProtected = errors.New("default campaign cannot be modified")
)

type Campaign struct {
	ID        uuid.UUID
	ClientID  uuid.UUID
	Name      string
	IsDefault bool
	CreatedAt time.Time
	UpdatedAt time.Time

	BackgroundStatus       string
	BackgroundPendingKey   string
	BackgroundThumbnailKey string
	BackgroundFilename     string
	BackgroundContentType  string

	events []event.DomainEvent
}

const DefaultCampaignName = "Default"

func NewCampaign(name string, clientID uuid.UUID) *Campaign {
	now := time.Now().UTC()
	c := &Campaign{
		ID:               uuid.New(),
		ClientID:         clientID,
		Name:             name,
		IsDefault:        false,
		CreatedAt:        now,
		UpdatedAt:        now,
		BackgroundStatus: BackgroundStatusNone,
	}
	c.recordEvent(CampaignCreated{
		ID:         uuid.New().String(),
		CampaignID: c.ID.String(),
		ClientID:   clientID.String(),
		Name:       c.Name,
		IsDefault:  false,
		Timestamp:  now,
	})
	return c
}

func NewDefaultCampaign(clientID uuid.UUID) *Campaign {
	now := time.Now().UTC()
	c := &Campaign{
		ID:               uuid.New(),
		ClientID:         clientID,
		Name:             DefaultCampaignName,
		IsDefault:        true,
		CreatedAt:        now,
		UpdatedAt:        now,
		BackgroundStatus: BackgroundStatusNone,
	}
	c.recordEvent(CampaignCreated{
		ID:         uuid.New().String(),
		CampaignID: c.ID.String(),
		ClientID:   clientID.String(),
		Name:       c.Name,
		IsDefault:  true,
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

func (c *Campaign) ApplyUpdate(name string) error {
	if c.IsDefault {
		return ErrDefaultCampaignProtected
	}
	c.Name = name
	c.UpdatedAt = time.Now().UTC()
	c.recordEvent(CampaignUpdated{
		ID:         uuid.New().String(),
		CampaignID: c.ID.String(),
		ClientID:   c.ClientID.String(),
		Name:       c.Name,
		Timestamp:  c.UpdatedAt,
	})
	return nil
}

func (c *Campaign) MarkDeleted() error {
	if c.IsDefault {
		return ErrDefaultCampaignProtected
	}
	c.recordEvent(CampaignDeleted{
		ID:         uuid.New().String(),
		CampaignID: c.ID.String(),
		ClientID:   c.ClientID.String(),
		Timestamp:  time.Now().UTC(),
	})
	return nil
}

func (c *Campaign) StartBackgroundUpload(pendingKey, filename, contentType string) {
	c.BackgroundStatus = BackgroundStatusPending
	c.BackgroundPendingKey = pendingKey
	c.BackgroundFilename = filename
	c.BackgroundContentType = contentType
	// Keep BackgroundThumbnailKey so the previous image stays available while replacing.
	c.UpdatedAt = time.Now().UTC()
	c.recordBackgroundUpdated()
}

func (c *Campaign) ApplyBackgroundReady(thumbnailKey string, contentType string) {
	c.BackgroundStatus = BackgroundStatusReady
	c.BackgroundPendingKey = ""
	c.BackgroundThumbnailKey = thumbnailKey
	if contentType != "" {
		c.BackgroundContentType = contentType
	}
	c.UpdatedAt = time.Now().UTC()
	c.recordBackgroundUpdated()
}

func (c *Campaign) MarkBackgroundFailed() {
	c.BackgroundStatus = BackgroundStatusFailed
	c.BackgroundPendingKey = ""
	c.UpdatedAt = time.Now().UTC()
	c.recordBackgroundUpdated()
}

func (c *Campaign) ClearBackground() {
	c.BackgroundStatus = BackgroundStatusNone
	c.BackgroundPendingKey = ""
	c.BackgroundThumbnailKey = ""
	c.BackgroundFilename = ""
	c.BackgroundContentType = ""
	c.UpdatedAt = time.Now().UTC()
	c.recordBackgroundUpdated()
}

func (c *Campaign) recordBackgroundUpdated() {
	c.recordEvent(CampaignBackgroundUpdated{
		ID:                     uuid.New().String(),
		CampaignID:             c.ID.String(),
		ClientID:               c.ClientID.String(),
		BackgroundStatus:       c.BackgroundStatus,
		BackgroundThumbnailKey: c.BackgroundThumbnailKey,
		Timestamp:              c.UpdatedAt,
	})
}
