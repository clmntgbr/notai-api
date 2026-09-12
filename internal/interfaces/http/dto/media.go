package dto

type PresignMediaRequest struct {
	Files []PresignMediaFileRequest `json:"files" validate:"required,min=1,max=20,dive"`
}

type PresignMediaFileRequest struct {
	Filename    string `json:"filename" validate:"required,min=1,max=255"`
	ContentType string `json:"contentType" validate:"required,min=1,max=127"`
}
