package presenter

import (
	"time"

	domainuser "go-api/internal/domain/user"
)

type UserDetailResponse struct {
	ID              string    `json:"id"`
	ClerkID         string    `json:"clerkId"`
	FirstName       *string   `json:"firstName"`
	LastName        *string   `json:"lastName"`
	Email           string    `json:"email"`
	CurrentClientID *string   `json:"currentClientId"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

func NewUserDetailResponseFromView(view domainuser.UserView) UserDetailResponse {
	return UserDetailResponse{
		ID:              view.ID.String(),
		ClerkID:         view.ClerkID,
		FirstName:       optionalNonEmptyString(view.FirstName),
		LastName:        optionalNonEmptyString(view.LastName),
		Email:           view.Email,
		CurrentClientID: optionalUUIDString(view.CurrentClientID),
		CreatedAt:       view.CreatedAt,
		UpdatedAt:       view.UpdatedAt,
	}
}

func NewUserDetailResponseFromEntity(user domainuser.User) UserDetailResponse {
	return UserDetailResponse{
		ID:              user.ID.String(),
		ClerkID:         user.ClerkID,
		FirstName:       optionalNonEmptyString(user.FirstName),
		LastName:        optionalNonEmptyString(user.LastName),
		Email:           user.Email,
		CurrentClientID: optionalUUIDString(user.CurrentClientID),
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	}
}
