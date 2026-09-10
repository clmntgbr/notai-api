package dto

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
	Name string `json:"name" validate:"required,min=1,max=255"`
}

type UpdateCampaignRequest struct {
	Name string `json:"name" validate:"required,min=1,max=255"`
}
