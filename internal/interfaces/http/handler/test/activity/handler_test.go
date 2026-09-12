package activitytest

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	queryactivity "go-api/internal/application/query/activity"
	domainactivity "go-api/internal/domain/activity"
	"go-api/internal/domain/paginate"
	"go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/testutil"

	"github.com/google/uuid"
)

type mockListActivityHandler struct {
	called bool
	query  queryactivity.ListByClientQuery
	views  []domainactivity.View
	total  int64
	err    error
}

func (m *mockListActivityHandler) Handle(
	_ context.Context,
	q queryactivity.ListByClientQuery,
) ([]domainactivity.View, int64, error) {
	m.called = true
	m.query = q
	return m.views, m.total, m.err
}

func newActivityHandler(list *mockListActivityHandler) *handler.ActivityHandler {
	if list == nil {
		list = &mockListActivityHandler{}
	}
	return handler.NewActivityHandler(list)
}

func mustJSONRequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()
	req, err := testutil.JSONRequest(method, path, body)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	return req
}

func TestActivityHandler_List_Success(t *testing.T) {
	id := uuid.MustParse("aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee")
	list := &mockListActivityHandler{
		views: []domainactivity.View{{
			ID:         id,
			ClientID:   testutil.TestClientID,
			Type:       domainactivity.TypeContentAIFlagged,
			ActorType:  domainactivity.ActorTypeSystem,
			ActorName:  domainactivity.ActorNameSystem,
			Message:    "“unboxing.png” flagged as AI-generated (score 78%)",
			Payload:    map[string]any{"filename": "unboxing.png"},
			OccurredAt: time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC),
		}},
		total: 1,
	}
	h := newActivityHandler(list)
	app := testutil.NewTestApp()
	app.Get("/activity", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.List)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/activity?page=1&limit=20", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if !list.called || list.query.ClientID != testutil.TestClientID {
		t.Fatalf("query: %+v", list.query)
	}
	if list.query.Query.SortBy != "occurred_at" || list.query.Query.OrderBy != paginate.OrderByDesc {
		t.Fatalf("defaults: %+v", list.query.Query)
	}
	body := testutil.DecodeJSONMap(t, resp)
	members, ok := body["members"].([]any)
	if !ok || len(members) != 1 {
		t.Fatalf("members: %#v", body["members"])
	}
	item, ok := members[0].(map[string]any)
	if !ok || item["type"] != domainactivity.TypeContentAIFlagged || item["actorName"] != "System" {
		t.Fatalf("item: %#v", members[0])
	}
}

func TestActivityHandler_List_Unauthorized(t *testing.T) {
	h := newActivityHandler(nil)
	app := testutil.NewTestApp()
	app.Get("/activity", h.List)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/activity", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestActivityHandler_List_MissingClient(t *testing.T) {
	h := newActivityHandler(nil)
	app := testutil.NewTestApp()
	app.Get("/activity", testutil.WithUserWithoutClient(testutil.TestUserID), h.List)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/activity", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestActivityHandler_List_Internal(t *testing.T) {
	list := &mockListActivityHandler{err: errors.New("boom")}
	h := newActivityHandler(list)
	app := testutil.NewTestApp()
	app.Get("/activity", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.List)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/activity", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	body := testutil.DecodeJSONMap(t, resp)
	if body["message"] != "Failed to list activity" {
		t.Fatalf("message: %#v", body["message"])
	}
}
