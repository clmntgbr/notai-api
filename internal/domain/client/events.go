package client

import "time"

const (
	EventTypeClientCreated       = "client.created.v1"
	EventTypeClientUpdated       = "client.updated.v1"
	EventTypeClientDeleted       = "client.deleted.v1"
	EventTypeClientMemberAdded   = "client.member_added.v1"
	EventTypeClientMemberRemoved = "client.member_removed.v1"
)

type ClientCreated struct {
	ID              string    `json:"eventId"`
	ClientID        string    `json:"clientId"`
	Name            string    `json:"name"`
	CreatedByUserID string    `json:"createdByUserId"`
	Timestamp       time.Time `json:"timestamp"`
}

func (e ClientCreated) EventID() string       { return e.ID }
func (e ClientCreated) EventType() string     { return EventTypeClientCreated }
func (e ClientCreated) AggregateID() string   { return e.ClientID }
func (e ClientCreated) OccurredAt() time.Time { return e.Timestamp }

type ClientUpdated struct {
	ID        string    `json:"eventId"`
	ClientID  string    `json:"clientId"`
	Name      string    `json:"name"`
	MemberIDs []string  `json:"memberIds"`
	Timestamp time.Time `json:"timestamp"`
}

func (e ClientUpdated) EventID() string       { return e.ID }
func (e ClientUpdated) EventType() string     { return EventTypeClientUpdated }
func (e ClientUpdated) AggregateID() string   { return e.ClientID }
func (e ClientUpdated) OccurredAt() time.Time { return e.Timestamp }

type ClientDeleted struct {
	ID        string    `json:"eventId"`
	ClientID  string    `json:"clientId"`
	MemberIDs []string  `json:"memberIds"`
	Timestamp time.Time `json:"timestamp"`
}

func (e ClientDeleted) EventID() string       { return e.ID }
func (e ClientDeleted) EventType() string     { return EventTypeClientDeleted }
func (e ClientDeleted) AggregateID() string   { return e.ClientID }
func (e ClientDeleted) OccurredAt() time.Time { return e.Timestamp }

type ClientMemberAdded struct {
	ID        string    `json:"eventId"`
	ClientID  string    `json:"clientId"`
	UserID    string    `json:"userId"`
	Timestamp time.Time `json:"timestamp"`
}

func (e ClientMemberAdded) EventID() string       { return e.ID }
func (e ClientMemberAdded) EventType() string     { return EventTypeClientMemberAdded }
func (e ClientMemberAdded) AggregateID() string   { return e.ClientID }
func (e ClientMemberAdded) OccurredAt() time.Time { return e.Timestamp }

type ClientMemberRemoved struct {
	ID        string    `json:"eventId"`
	ClientID  string    `json:"clientId"`
	UserID    string    `json:"userId"`
	Timestamp time.Time `json:"timestamp"`
}

func (e ClientMemberRemoved) EventID() string       { return e.ID }
func (e ClientMemberRemoved) EventType() string     { return EventTypeClientMemberRemoved }
func (e ClientMemberRemoved) AggregateID() string   { return e.ClientID }
func (e ClientMemberRemoved) OccurredAt() time.Time { return e.Timestamp }
