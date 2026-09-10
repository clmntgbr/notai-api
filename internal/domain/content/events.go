package content

import "time"

const (
	EventTypeContentCreated       = "content.created.v1"
	EventTypeContentUploaded      = "content.uploaded.v1"
	EventTypeContentStatusChanged = "content.status_changed.v1"
)

type ContentCreated struct {
	ID          string    `json:"eventId"`
	ContentID   string    `json:"contentId"`
	CampaignID  string    `json:"campaignId"`
	ClientID    string    `json:"clientId"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"contentType"`
	ObjectKey   string    `json:"objectKey"`
	Status      string    `json:"status"`
	Timestamp   time.Time `json:"timestamp"`
}

func (e ContentCreated) EventID() string       { return e.ID }
func (e ContentCreated) EventType() string     { return EventTypeContentCreated }
func (e ContentCreated) AggregateID() string   { return e.ContentID }
func (e ContentCreated) OccurredAt() time.Time { return e.Timestamp }

type ContentUploaded struct {
	ID           string    `json:"eventId"`
	ContentID    string    `json:"contentId"`
	CampaignID   string    `json:"campaignId"`
	ClientID     string    `json:"clientId"`
	ObjectKey    string    `json:"objectKey"`
	ThumbnailKey string    `json:"thumbnailKey"`
	SizeBytes    int64     `json:"sizeBytes"`
	Timestamp    time.Time `json:"timestamp"`
}

func (e ContentUploaded) EventID() string       { return e.ID }
func (e ContentUploaded) EventType() string     { return EventTypeContentUploaded }
func (e ContentUploaded) AggregateID() string   { return e.ContentID }
func (e ContentUploaded) OccurredAt() time.Time { return e.Timestamp }

type ContentStatusChanged struct {
	ID         string    `json:"eventId"`
	ContentID  string    `json:"contentId"`
	CampaignID string    `json:"campaignId"`
	ClientID   string    `json:"clientId"`
	Status     string    `json:"status"`
	Timestamp  time.Time `json:"timestamp"`
}

func (e ContentStatusChanged) EventID() string       { return e.ID }
func (e ContentStatusChanged) EventType() string     { return EventTypeContentStatusChanged }
func (e ContentStatusChanged) AggregateID() string   { return e.ContentID }
func (e ContentStatusChanged) OccurredAt() time.Time { return e.Timestamp }
