package media

import "time"

const (
	EventTypeMediaUploadRequested   = "media.upload_requested.v1"
	EventTypeMediaUploaded          = "media.uploaded.v1"
	EventTypeMediaProcessingStarted = "media.processing_started.v1"
	EventTypeMediaStatusChanged     = "media.status_changed.v1"
	EventTypeMediaVerdictRendered   = "media.verdict_rendered.v1"
)

type MediaUploadRequested struct {
	ID          string    `json:"eventId"`
	MediaID     string    `json:"mediaId"`
	CampaignID  string    `json:"campaignId"`
	ClientID    string    `json:"clientId"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"contentType"`
	MediaType   string    `json:"mediaType"`
	ObjectKey   string    `json:"objectKey"`
	Status      string    `json:"status"`
	Timestamp   time.Time `json:"timestamp"`
}

func (e MediaUploadRequested) EventID() string       { return e.ID }
func (e MediaUploadRequested) EventType() string     { return EventTypeMediaUploadRequested }
func (e MediaUploadRequested) AggregateID() string   { return e.MediaID }
func (e MediaUploadRequested) OccurredAt() time.Time { return e.Timestamp }

type MediaUploaded struct {
	ID         string    `json:"eventId"`
	MediaID    string    `json:"mediaId"`
	CampaignID string    `json:"campaignId"`
	ClientID   string    `json:"clientId"`
	MediaType  string    `json:"mediaType"`
	ObjectKey  string    `json:"objectKey"`
	SizeBytes  int64     `json:"sizeBytes"`
	Timestamp  time.Time `json:"timestamp"`
}

func (e MediaUploaded) EventID() string       { return e.ID }
func (e MediaUploaded) EventType() string     { return EventTypeMediaUploaded }
func (e MediaUploaded) AggregateID() string   { return e.MediaID }
func (e MediaUploaded) OccurredAt() time.Time { return e.Timestamp }

type MediaProcessingStarted struct {
	ID         string    `json:"eventId"`
	MediaID    string    `json:"mediaId"`
	CampaignID string    `json:"campaignId"`
	ClientID   string    `json:"clientId"`
	Timestamp  time.Time `json:"timestamp"`
}

func (e MediaProcessingStarted) EventID() string       { return e.ID }
func (e MediaProcessingStarted) EventType() string     { return EventTypeMediaProcessingStarted }
func (e MediaProcessingStarted) AggregateID() string   { return e.MediaID }
func (e MediaProcessingStarted) OccurredAt() time.Time { return e.Timestamp }

type MediaStatusChanged struct {
	ID         string    `json:"eventId"`
	MediaID    string    `json:"mediaId"`
	CampaignID string    `json:"campaignId"`
	ClientID   string    `json:"clientId"`
	Status     string    `json:"status"`
	Label      string    `json:"label,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

func (e MediaStatusChanged) EventID() string       { return e.ID }
func (e MediaStatusChanged) EventType() string     { return EventTypeMediaStatusChanged }
func (e MediaStatusChanged) AggregateID() string   { return e.MediaID }
func (e MediaStatusChanged) OccurredAt() time.Time { return e.Timestamp }

type MediaVerdictRendered struct {
	ID           string    `json:"eventId"`
	MediaID      string    `json:"mediaId"`
	CampaignID   string    `json:"campaignId"`
	ClientID     string    `json:"clientId"`
	Label        string    `json:"label"`
	FlaggedCount int       `json:"flaggedCount"`
	TotalCount   int       `json:"totalCount"`
	FailedCount  int       `json:"failedCount"`
	Timestamp    time.Time `json:"timestamp"`
}

func (e MediaVerdictRendered) EventID() string       { return e.ID }
func (e MediaVerdictRendered) EventType() string     { return EventTypeMediaVerdictRendered }
func (e MediaVerdictRendered) AggregateID() string   { return e.MediaID }
func (e MediaVerdictRendered) OccurredAt() time.Time { return e.Timestamp }
