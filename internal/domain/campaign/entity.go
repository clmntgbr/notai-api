package campaign

import (
	"errors"
	"time"

	"go-api/internal/domain/event"

	"github.com/google/uuid"
)

var (
	ErrDefaultCampaignProtected = errors.New("default campaign cannot be modified")
	ErrInvalidSchedule          = errors.New("endAt must be after startAt")
)

type Campaign struct {
	ID        uuid.UUID
	ClientID  uuid.UUID
	Name      string
	IsDefault bool
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
	StartAt *time.Time
	EndAt   *time.Time

	BackgroundStatus       string
	BackgroundPendingKey   string
	BackgroundThumbnailKey string
	BackgroundFilename     string
	BackgroundContentType  string

	events []event.DomainEvent
}

const DefaultCampaignName = "Default"

func ValidateSchedule(startAt, endAt *time.Time) error {
	if startAt != nil && endAt != nil && endAt.Before(*startAt) {
		return ErrInvalidSchedule
	}
	return nil
}

func NewCampaign(name string, clientID uuid.UUID, startAt, endAt *time.Time) (*Campaign, error) {
	if err := ValidateSchedule(startAt, endAt); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	c := &Campaign{
		ID:               uuid.New(),
		ClientID:         clientID,
		Name:             name,
		IsDefault:        false,
		CreatedAt:        now,
		UpdatedAt:        now,
		StartAt:        cloneTimePtr(startAt),
		EndAt:          cloneTimePtr(endAt),
		BackgroundStatus: BackgroundStatusNone,
	}
	c.recordEvent(CampaignCreated{
		ID:         uuid.New().String(),
		CampaignID: c.ID.String(),
		ClientID:   clientID.String(),
		Name:       c.Name,
		IsDefault:  false,
		StartAt:  timePtrValue(c.StartAt),
		EndAt:    timePtrValue(c.EndAt),
		Timestamp:  now,
	})
	return c, nil
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

func (c *Campaign) ApplyUpdate(name string, startAt, endAt *time.Time) error {
	if c.IsDefault {
		return ErrDefaultCampaignProtected
	}
	if c.IsDeleted() {
		return errors.New("campaign not found")
	}
	if err := ValidateSchedule(startAt, endAt); err != nil {
		return err
	}
	c.Name = name
	c.StartAt = cloneTimePtr(startAt)
	c.EndAt = cloneTimePtr(endAt)
	c.UpdatedAt = time.Now().UTC()
	c.recordEvent(CampaignUpdated{
		ID:         uuid.New().String(),
		CampaignID: c.ID.String(),
		ClientID:   c.ClientID.String(),
		Name:       c.Name,
		StartAt:  timePtrValue(c.StartAt),
		EndAt:    timePtrValue(c.EndAt),
		Timestamp:  c.UpdatedAt,
	})
	return nil
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copied := value.UTC()
	return &copied
}

func timePtrValue(value *time.Time) *time.Time {
	return cloneTimePtr(value)
}

func (c *Campaign) IsDeleted() bool {
	return c.DeletedAt != nil
}

func (c *Campaign) SoftDelete() error {
	if c.IsDefault {
		return ErrDefaultCampaignProtected
	}
	if c.IsDeleted() {
		return nil
	}
	now := time.Now().UTC()
	c.DeletedAt = &now
	c.UpdatedAt = now
	c.recordEvent(CampaignDeleted{
		ID:         uuid.New().String(),
		CampaignID: c.ID.String(),
		ClientID:   c.ClientID.String(),
		Timestamp:  now,
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
