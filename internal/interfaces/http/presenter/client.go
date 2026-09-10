package presenter

import (
	"time"

	domainclient "go-api/internal/domain/client"

	"github.com/google/uuid"
)

type ClientDetailResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"isActive"`
	MemberIDs []string  `json:"memberIds"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func NewClientDetailResponseFromView(
	view domainclient.ClientView,
	currentClientID *uuid.UUID,
) ClientDetailResponse {
	return ClientDetailResponse{
		ID:        view.ID.String(),
		Name:      view.Name,
		IsActive:  isCurrentClient(view.ID, currentClientID),
		MemberIDs: uuidStrings(view.MemberIDs),
		CreatedAt: view.CreatedAt,
		UpdatedAt: view.UpdatedAt,
	}
}

func NewClientDetailResponseFromEntity(
	client domainclient.Client,
	currentClientID *uuid.UUID,
) ClientDetailResponse {
	return ClientDetailResponse{
		ID:        client.ID.String(),
		Name:      client.Name,
		IsActive:  isCurrentClient(client.ID, currentClientID),
		MemberIDs: uuidStrings(client.MemberIDs),
		CreatedAt: client.CreatedAt,
		UpdatedAt: client.UpdatedAt,
	}
}

func NewClientListResponseFromViews(
	views []domainclient.ClientView,
	currentClientID *uuid.UUID,
) []ClientDetailResponse {
	items := make([]ClientDetailResponse, 0, len(views))
	for _, view := range views {
		items = append(items, NewClientDetailResponseFromView(view, currentClientID))
	}
	return items
}

func isCurrentClient(clientID uuid.UUID, currentClientID *uuid.UUID) bool {
	return currentClientID != nil && *currentClientID == clientID
}

func uuidStrings(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}
