package dto

import "time"

type CreateClientRequest struct {
	Name string `json:"name" validate:"required,min=1,max=255"`
}

type UpdateClientRequest struct {
	Name string `json:"name" validate:"required,min=1,max=255"`
}

type SetCurrentClientRequest struct {
	ClientID string `json:"clientId" validate:"required,uuid"`
}

type CreateCampaignRequest struct {
	Name      string     `json:"name" validate:"required,min=1,max=255"`
	StartAt *time.Time `json:"startAt"`
	EndAt   *time.Time `json:"endAt"`
}

type UpdateCampaignRequest struct {
	Name      string     `json:"name" validate:"required,min=1,max=255"`
	StartAt *time.Time `json:"startAt"`
	EndAt   *time.Time `json:"endAt"`
}
