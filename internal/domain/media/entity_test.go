package media_test

import (
	"testing"

	domainmedia "go-api/internal/domain/media"

	"github.com/google/uuid"
)

func TestIsMediaObjectKey(t *testing.T) {
	key := domainmedia.NewObjectKey(uuid.New(), uuid.New(), uuid.New(), "a.png")
	if !domainmedia.IsMediaObjectKey(key) {
		t.Fatalf("expected media key: %s", key)
	}
	frame := domainmedia.NewFrameObjectKey(uuid.New(), uuid.New(), uuid.New(), 3)
	if domainmedia.IsMediaObjectKey(frame) {
		t.Fatalf("frame should not be media original: %s", frame)
	}
	if !domainmedia.IsFrameObjectKey(frame) {
		t.Fatalf("expected frame key: %s", frame)
	}
}

func TestDetectMediaType(t *testing.T) {
	mt, err := domainmedia.DetectMediaType("clip.mp4", "video/mp4")
	if err != nil || mt != domainmedia.MediaTypeVideo {
		t.Fatalf("got %s %v", mt, err)
	}
	mt, err = domainmedia.DetectMediaType("shot.jpg", "image/jpeg")
	if err != nil || mt != domainmedia.MediaTypeImage {
		t.Fatalf("got %s %v", mt, err)
	}
}
