package campaign

import "time"

const (
	EventTypeCampaignCreated           = "campaign.created.v1"
	EventTypeCampaignUpdated           = "campaign.updated.v1"
	EventTypeCampaignDeleted           = "campaign.deleted.v1"
	EventTypeCampaignBackgroundUpdated = "campaign.background_updated.v1"
)

type CampaignCreated struct {
	ID         string     `json:"eventId"`
	CampaignID string     `json:"campaignId"`
	ClientID   string     `json:"clientId"`
	Name       string     `json:"name"`
	IsDefault  bool       `json:"isDefault"`
	StartAt  *time.Time `json:"startAt,omitempty"`
	EndAt    *time.Time `json:"endAt,omitempty"`
	Timestamp  time.Time  `json:"timestamp"`
}

func (e CampaignCreated) EventID() string       { return e.ID }
func (e CampaignCreated) EventType() string     { return EventTypeCampaignCreated }
func (e CampaignCreated) AggregateID() string   { return e.CampaignID }
func (e CampaignCreated) OccurredAt() time.Time { return e.Timestamp }

type CampaignUpdated struct {
	ID         string     `json:"eventId"`
	CampaignID string     `json:"campaignId"`
	ClientID   string     `json:"clientId"`
	Name       string     `json:"name"`
	StartAt  *time.Time `json:"startAt,omitempty"`
	EndAt    *time.Time `json:"endAt,omitempty"`
	Timestamp  time.Time  `json:"timestamp"`
}

func (e CampaignUpdated) EventID() string       { return e.ID }
func (e CampaignUpdated) EventType() string     { return EventTypeCampaignUpdated }
func (e CampaignUpdated) AggregateID() string   { return e.CampaignID }
func (e CampaignUpdated) OccurredAt() time.Time { return e.Timestamp }

type CampaignDeleted struct {
	ID         string    `json:"eventId"`
	CampaignID string    `json:"campaignId"`
	ClientID   string    `json:"clientId"`
	Timestamp  time.Time `json:"timestamp"`
}

func (e CampaignDeleted) EventID() string       { return e.ID }
func (e CampaignDeleted) EventType() string     { return EventTypeCampaignDeleted }
func (e CampaignDeleted) AggregateID() string   { return e.CampaignID }
func (e CampaignDeleted) OccurredAt() time.Time { return e.Timestamp }

type CampaignBackgroundUpdated struct {
	ID                     string    `json:"eventId"`
	CampaignID             string    `json:"campaignId"`
	ClientID               string    `json:"clientId"`
	BackgroundStatus       string    `json:"backgroundStatus"`
	BackgroundThumbnailKey string    `json:"backgroundThumbnailKey"`
	Timestamp              time.Time `json:"timestamp"`
}

func (e CampaignBackgroundUpdated) EventID() string       { return e.ID }
func (e CampaignBackgroundUpdated) EventType() string     { return EventTypeCampaignBackgroundUpdated }
func (e CampaignBackgroundUpdated) AggregateID() string   { return e.CampaignID }
func (e CampaignBackgroundUpdated) OccurredAt() time.Time { return e.Timestamp }
