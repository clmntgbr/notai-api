package dto

type PresignContentsRequest struct {
	Files []PresignContentFileRequest `json:"files" validate:"required,min=1,max=20,dive"`
}

type PresignContentFileRequest struct {
	Filename    string `json:"filename" validate:"required,min=1,max=255"`
	ContentType string `json:"contentType" validate:"omitempty,max=127"`
}

type PresignContentsResponse struct {
	Items []PresignContentItemResponse `json:"items"`
}

type PresignContentItemResponse struct {
	ContentID string `json:"contentId"`
	URL       string `json:"url"`
	ObjectKey string `json:"objectKey"`
	Filename  string `json:"filename"`
}
