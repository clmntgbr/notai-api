package dto

type PresignContentsRequest struct {
	CampaignID string                      `json:"campaignId" validate:"omitempty,uuid"`
	Files      []PresignContentFileRequest `json:"files" validate:"required,min=1,max=20,dive"`
}

type PresignContentFileRequest struct {
	Filename    string `json:"filename" validate:"required,min=1,max=255"`
	ContentType string `json:"contentType" validate:"omitempty,max=127"`
}
