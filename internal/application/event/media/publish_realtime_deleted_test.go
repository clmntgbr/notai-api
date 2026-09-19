package media

import (
	"context"
	"encoding/json"
	"testing"

	domainclient "go-api/internal/domain/client"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/paginate"

	"github.com/google/uuid"
)

type memRealtime struct {
	published int
}

func (m *memRealtime) PublishToUserInterest(
	_ context.Context,
	_ uuid.UUID,
	_ string,
	_ string,
	_ any,
) error {
	m.published++
	return nil
}

type memClientRepo struct {
	view *domainclient.ClientView
}

func (r *memClientRepo) FindByID(context.Context, uuid.UUID) (*domainclient.ClientView, error) {
	return r.view, nil
}
func (r *memClientRepo) FindByUserID(context.Context, uuid.UUID) ([]domainclient.ClientView, error) {
	return nil, nil
}
func (r *memClientRepo) FindPageByUserID(
	context.Context,
	uuid.UUID,
	paginate.PaginateQuery,
) ([]domainclient.ClientView, int64, error) {
	return nil, 0, nil
}

func TestPublishRealtimeHandler_OnDeleted_CascadeSkipsPublish(t *testing.T) {
	rt := &memRealtime{}
	clientID := uuid.New()
	h := NewPublishRealtimeHandler(rt, &memClientRepo{
		view: &domainclient.ClientView{
			ID:        clientID,
			MemberIDs: []uuid.UUID{uuid.New()},
		},
	})

	payload, err := json.Marshal(domainmedia.MediaDeleted{
		ID:       "evt-1",
		MediaID:  uuid.New().String(),
		ClientID: clientID.String(),
		Cascade:  true,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := h.OnDeleted(context.Background(), payload); err != nil {
		t.Fatalf("OnDeleted: %v", err)
	}
	if rt.published != 0 {
		t.Fatalf("cascade must not publish realtime, got %d", rt.published)
	}
}

func TestPublishRealtimeHandler_OnDeleted_StandalonePublishes(t *testing.T) {
	rt := &memRealtime{}
	clientID := uuid.New()
	h := NewPublishRealtimeHandler(rt, &memClientRepo{
		view: &domainclient.ClientView{
			ID:        clientID,
			MemberIDs: []uuid.UUID{uuid.New(), uuid.New()},
		},
	})

	payload, err := json.Marshal(domainmedia.MediaDeleted{
		ID:       "evt-1",
		MediaID:  uuid.New().String(),
		ClientID: clientID.String(),
		Cascade:  false,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := h.OnDeleted(context.Background(), payload); err != nil {
		t.Fatalf("OnDeleted: %v", err)
	}
	if rt.published != 2 {
		t.Fatalf("standalone delete must publish to members, got %d", rt.published)
	}
}
