package workspacetest

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	queryworkspace "go-api/internal/application/query/workspace"
	domainworkspace "go-api/internal/domain/workspace"
	"go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/presenter"
	"go-api/internal/interfaces/http/testutil"

	"github.com/google/uuid"
)

type mockGetOwnedWorkspaceHandler struct {
	called bool
	query  queryworkspace.GetOwnedWorkspaceQuery
	view   *domainworkspace.WorkspaceView
	err    error
}

func (m *mockGetOwnedWorkspaceHandler) Handle(
	_ context.Context,
	q queryworkspace.GetOwnedWorkspaceQuery,
) (*domainworkspace.WorkspaceView, error) {
	m.called = true
	m.query = q
	return m.view, m.err
}

func sampleWorkspaceView() *domainworkspace.WorkspaceView {
	subID := uuid.MustParse("01960000-0000-7000-8000-00000000000d")
	return &domainworkspace.WorkspaceView{
		ID:             testutil.TestWorkspaceID,
		Name:           "Personal Workspace",
		OwnerUserID:    testutil.TestUserID,
		SubscriptionID: &subID,
		CreatedAt:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestWorkspaceHandler_GetMe_Success(t *testing.T) {
	getOwned := &mockGetOwnedWorkspaceHandler{view: sampleWorkspaceView()}
	h := handler.NewWorkspaceHandler(getOwned)

	app := testutil.NewTestApp()
	app.Get("/workspaces/me", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetMe)

	req, err := testutil.JSONRequest(http.MethodGet, "/workspaces/me", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusOK)
	}
	if !getOwned.called || getOwned.query.OwnerUserID != testutil.TestUserID {
		t.Fatalf("unexpected query: %+v", getOwned.query)
	}

	var out presenter.WorkspaceResponse
	testutil.DecodeJSON(t, resp, &out)
	if out.ID != testutil.TestWorkspaceID.String() {
		t.Fatalf("id: got %s", out.ID)
	}
	if out.OwnerUserID != testutil.TestUserID.String() {
		t.Fatalf("owner: got %s", out.OwnerUserID)
	}
}

func TestWorkspaceHandler_GetMe_Unauthorized(t *testing.T) {
	h := handler.NewWorkspaceHandler(&mockGetOwnedWorkspaceHandler{})
	app := testutil.NewTestApp()
	app.Get("/workspaces/me", h.GetMe)

	req, _ := testutil.JSONRequest(http.MethodGet, "/workspaces/me", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestWorkspaceHandler_GetMe_NotFound(t *testing.T) {
	getOwned := &mockGetOwnedWorkspaceHandler{err: queryworkspace.ErrWorkspaceNotFound}
	h := handler.NewWorkspaceHandler(getOwned)

	app := testutil.NewTestApp()
	app.Get("/workspaces/me", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetMe)

	req, _ := testutil.JSONRequest(http.MethodGet, "/workspaces/me", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestWorkspaceHandler_GetMe_HandlerError_Internal(t *testing.T) {
	getOwned := &mockGetOwnedWorkspaceHandler{err: errors.New("db down")}
	h := handler.NewWorkspaceHandler(getOwned)

	app := testutil.NewTestApp()
	app.Get("/workspaces/me", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetMe)

	req, _ := testutil.JSONRequest(http.MethodGet, "/workspaces/me", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}
