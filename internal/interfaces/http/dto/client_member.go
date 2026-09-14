package dto

type AddClientMemberRequest struct {
	UserID string `json:"userId" validate:"required,uuid"`
}
