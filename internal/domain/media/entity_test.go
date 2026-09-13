package media_test

import (
	"testing"

	domaincontent "go-api/internal/domain/content"
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

func TestMarkUploaded_EmitsSingleUploadedEvent(t *testing.T) {
	m, err := domainmedia.NewPendingUpload(uuid.New(), uuid.New(), "a.png", "image/png")
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	_ = m.PullEvents()

	if err := m.MarkUploaded(12, "image/png"); err != nil {
		t.Fatalf("upload: %v", err)
	}
	events := m.PullEvents()
	if len(events) != 1 {
		t.Fatalf("events: got %d %#v", len(events), events)
	}
	if events[0].EventType() != domainmedia.EventTypeMediaUploaded {
		t.Fatalf("type: %s", events[0].EventType())
	}
}

func TestRenderGlobalVerdict_EmitsSingleVerdictEvent(t *testing.T) {
	m, err := domainmedia.NewPendingUpload(uuid.New(), uuid.New(), "a.png", "image/png")
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	_ = m.PullEvents()
	if err := m.MarkUploaded(12, "image/png"); err != nil {
		t.Fatalf("upload: %v", err)
	}
	_ = m.PullEvents()
	if err := m.StartProcessing(); err != nil {
		t.Fatalf("processing: %v", err)
	}
	_ = m.PullEvents()

	if err := m.RenderGlobalVerdict(domainmedia.Verdict{
		Label:      domaincontent.LabelHuman,
		TotalCount: 1,
	}); err != nil {
		t.Fatalf("verdict: %v", err)
	}
	events := m.PullEvents()
	if len(events) != 1 {
		t.Fatalf("events: got %d %#v", len(events), events)
	}
	if events[0].EventType() != domainmedia.EventTypeMediaVerdictRendered {
		t.Fatalf("type: %s", events[0].EventType())
	}
}
