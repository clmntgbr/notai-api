package campaign

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

var ErrUnsupportedBackgroundType = errors.New("unsupported background file type")

const (
	BackgroundStatusNone    = "none"
	BackgroundStatusPending = "pending"
	BackgroundStatusReady   = "ready"
	BackgroundStatusFailed  = "failed"

	MaxBackgroundBytes = 5 << 20 // 5 MiB
	ThumbnailMaxWidth  = 400
)

var imageExtensions = map[string]struct{}{
	"jpg":  {},
	"jpeg": {},
	"png":  {},
	"webp": {},
}

func FileExtension(filename string) string {
	return strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
}

func ValidateBackgroundFilename(filename string) error {
	ext := FileExtension(filename)
	if ext == "" {
		return ErrUnsupportedBackgroundType
	}
	if _, ok := imageExtensions[ext]; !ok {
		return ErrUnsupportedBackgroundType
	}
	return nil
}

func IsImageFilename(filename string) bool {
	_, ok := imageExtensions[FileExtension(filename)]
	return ok
}

func NewBackgroundObjectKey(clientID, campaignID uuid.UUID, filename string) string {
	ext := FileExtension(filename)
	fileKey := uuid.New().String()
	if ext != "" {
		fileKey += "." + ext
	}
	return fmt.Sprintf("clients/%s/campaigns/%s/%s", clientID.String(), campaignID.String(), fileKey)
}

func NewBackgroundThumbnailKey(clientID, campaignID uuid.UUID) string {
	return fmt.Sprintf(
		"clients/%s/campaigns/%s/thumbnails/%s.jpg",
		clientID.String(),
		campaignID.String(),
		campaignID.String(),
	)
}

func DecodeObjectKey(raw string) (string, error) {
	decoded, err := url.QueryUnescape(raw)
	if err != nil {
		return "", err
	}
	return decoded, nil
}

func IsThumbnailObjectKey(key string) bool {
	return strings.Contains(key, "/thumbnails/")
}
