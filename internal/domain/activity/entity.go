package activity

import (
	"time"

	"github.com/google/uuid"
)

const (
	TypeContentAIFlagged     = "content.ai_flagged"
	TypeContentManualReview  = "content.manual_review"
	TypeContentHumanVerified = "content.human_verified"
	TypeContentFailed        = "content.failed"
	TypeCampaignCreated      = "campaign.created"
	TypeClientMemberAdded    = "client.member_added"

	ActorTypeSystem = "system"
	ActorTypeUser   = "user"

	ActorNameSystem = "System"
)

// Event is a denormalized activity-feed projection row.
type Event struct {
	ID          uuid.UUID
	ClientID    uuid.UUID
	Type        string
	ActorType   string
	ActorUserID *uuid.UUID
	ActorName   string
	Message     string
	Payload     map[string]any
	OccurredAt  time.Time
	CreatedAt   time.Time
}

// View is the read-side shape returned by queries.
type View struct {
	ID          uuid.UUID
	ClientID    uuid.UUID
	Type        string
	ActorType   string
	ActorUserID *uuid.UUID
	ActorName   string
	Message     string
	Payload     map[string]any
	OccurredAt  time.Time
}
