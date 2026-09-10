package clienttest

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	clientcmd "go-api/internal/application/command/client"
	queryclient "go-api/internal/application/query/client"
	domainclient "go-api/internal/domain/client"
	"go-api/internal/domain/paginate"
	"go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/presenter"
	"go-api/internal/interfaces/http/testutil"

	"github.com/google/uuid"
)

type mockCreateClientHandler struct {
	called bool
	cmd    clientcmd.CreateClientCommand
	result *domainclient.Client
	err    error
}

func (m *mockCreateClientHandler) Handle(
	_ context.Context,
	cmd clientcmd.CreateClientCommand,
) (*domainclient.Client, error) {
	m.called = true
	m.cmd = cmd
	return m.result, m.err
}

type mockUpdateClientHandler struct {
	called bool
	cmd    clientcmd.UpdateClientCommand
	err    error
}

func (m *mockUpdateClientHandler) Handle(_ context.Context, cmd clientcmd.UpdateClientCommand) error {
	m.called = true
	m.cmd = cmd
	return m.err
}

type mockDeleteClientHandler struct {
	called bool
	id     uuid.UUID
	err    error
}

func (m *mockDeleteClientHandler) Handle(_ context.Context, id uuid.UUID) error {
	m.called = true
	m.id = id
	return m.err
}

type mockRemoveMemberHandler struct {
	called bool
	cmd    clientcmd.RemoveClientMemberCommand
	err    error
}

func (m *mockRemoveMemberHandler) Handle(_ context.Context, cmd clientcmd.RemoveClientMemberCommand) error {
	m.called = true
	m.cmd = cmd
	return m.err
}

type mockGetClientByIDHandler struct {
	calls int
	views []*domainclient.ClientView
	errs  []error
}

func (m *mockGetClientByIDHandler) Handle(
	_ context.Context,
	_ queryclient.GetClientByIDQuery,
) (*domainclient.ClientView, error) {
	idx := m.calls
	m.calls++
	if idx >= len(m.errs) {
		return nil, errors.New("unexpected get by id call")
	}
	if m.errs[idx] != nil {
		return nil, m.errs[idx]
	}
	return m.views[idx], nil
}

type mockListClientsByUserHandler struct {
	called bool
	query  queryclient.ListClientsByUserQuery
	views  []domainclient.ClientView
	total  int64
	err    error
}

func (m *mockListClientsByUserHandler) Handle(
	_ context.Context,
	q queryclient.ListClientsByUserQuery,
) ([]domainclient.ClientView, int64, error) {
	m.called = true
	m.query = q
	return m.views, m.total, m.err
}

func newClientHandler(create *mockCreateClientHandler, update *mockUpdateClientHandler, deleteH *mockDeleteClientHandler, remove *mockRemoveMemberHandler, getByID *mockGetClientByIDHandler, list *mockListClientsByUserHandler) *handler.ClientHandler {
	if create == nil {
		create = &mockCreateClientHandler{}
	}
	if update == nil {
		update = &mockUpdateClientHandler{}
	}
	if deleteH == nil {
		deleteH = &mockDeleteClientHandler{}
	}
	if remove == nil {
		remove = &mockRemoveMemberHandler{}
	}
	if getByID == nil {
		getByID = &mockGetClientByIDHandler{}
	}
	if list == nil {
		list = &mockListClientsByUserHandler{}
	}
	return handler.NewClientHandler(create, update, deleteH, remove, getByID, list)
}

func sampleClientEntity() *domainclient.Client {
	return &domainclient.Client{
		ID:        testutil.TestClientID,
		Name:      "My Client",
		MemberIDs: []uuid.UUID{testutil.TestUserID},
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func sampleClientView() *domainclient.ClientView {
	e := sampleClientEntity()
	return &domainclient.ClientView{
		ID:        e.ID,
		Name:      e.Name,
		MemberIDs: e.MemberIDs,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

func mustJSONRequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()
	req, err := testutil.JSONRequest(method, path, body)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	return req
}

func TestClientHandler_List_Success(t *testing.T) {
	view := sampleClientView()
	list := &mockListClientsByUserHandler{views: []domainclient.ClientView{*view}, total: 1}
	h := newClientHandler(nil, nil, nil, nil, nil, list)

	app := testutil.NewTestApp()
	app.Get("/clients", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.List)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/clients?page=2&limit=10&search=my", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusOK)
	}
	if !list.called {
		t.Fatal("list handler not called")
	}
	if list.query.UserID != testutil.TestUserID {
		t.Fatalf("user id: got %s", list.query.UserID)
	}
	if list.query.Query.Page != 2 || list.query.Query.Limit != 10 {
		t.Fatalf("pagination: page=%d limit=%d", list.query.Query.Page, list.query.Query.Limit)
	}
	if list.query.Query.SortBy != "created_at" || list.query.Query.OrderBy != paginate.OrderByAsc {
		t.Fatalf("sort: %s %s", list.query.Query.SortBy, list.query.Query.OrderBy)
	}

	var out struct {
		Members []presenter.ClientDetailResponse `json:"members"`
	}
	testutil.DecodeJSON(t, resp, &out)
	if len(out.Members) != 1 || out.Members[0].ID != testutil.TestClientID.String() {
		t.Fatalf("members: %+v", out.Members)
	}
	if !out.Members[0].IsActive {
		t.Fatal("expected isActive true")
	}
}

func TestClientHandler_List_Unauthorized(t *testing.T) {
	h := newClientHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/clients", h.List)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/clients", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestClientHandler_Create_Success(t *testing.T) {
	create := &mockCreateClientHandler{result: sampleClientEntity()}
	h := newClientHandler(create, nil, nil, nil, nil, nil)

	app := testutil.NewTestApp()
	app.Post("/clients", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Create)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost, "/clients", map[string]any{"name": "My Client"}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusCreated)
	}
	if !create.called || create.cmd.Name != "My Client" || create.cmd.CreatorUserID != testutil.TestUserID {
		t.Fatalf("create cmd: %+v", create.cmd)
	}

	var out presenter.ClientDetailResponse
	testutil.DecodeJSON(t, resp, &out)
	if !out.IsActive || out.ID != testutil.TestClientID.String() {
		t.Fatalf("response: %+v", out)
	}
}

func TestClientHandler_Create_Unauthorized(t *testing.T) {
	h := newClientHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post("/clients", h.Create)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost, "/clients", map[string]any{"name": "x"}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestClientHandler_Create_InvalidInput(t *testing.T) {
	create := &mockCreateClientHandler{}
	h := newClientHandler(create, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post("/clients", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Create)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost, "/clients", map[string]any{"name": ""}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if create.called {
		t.Fatal("create must not be called")
	}
}

func TestClientHandler_Create_HandlerError_Internal(t *testing.T) {
	create := &mockCreateClientHandler{err: errors.New("boom")}
	h := newClientHandler(create, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post("/clients", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Create)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost, "/clients", map[string]any{"name": "My Client"}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestClientHandler_GetByID_Success(t *testing.T) {
	getByID := &mockGetClientByIDHandler{views: []*domainclient.ClientView{sampleClientView()}, errs: []error{nil}}
	h := newClientHandler(nil, nil, nil, nil, getByID, nil)
	app := testutil.NewTestApp()
	app.Get("/clients/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetByID)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/clients/"+testutil.TestClientID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestClientHandler_GetByID_Unauthorized(t *testing.T) {
	h := newClientHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/clients/:id", h.GetByID)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/clients/"+testutil.TestClientID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestClientHandler_GetByID_InvalidID(t *testing.T) {
	h := newClientHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/clients/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetByID)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/clients/not-a-uuid", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestClientHandler_GetByID_NotFound(t *testing.T) {
	getByID := &mockGetClientByIDHandler{views: []*domainclient.ClientView{nil}, errs: []error{errors.New("client not found")}}
	h := newClientHandler(nil, nil, nil, nil, getByID, nil)
	app := testutil.NewTestApp()
	app.Get("/clients/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetByID)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/clients/"+testutil.TestClientID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestClientHandler_Update_Success(t *testing.T) {
	update := &mockUpdateClientHandler{}
	getByID := &mockGetClientByIDHandler{views: []*domainclient.ClientView{sampleClientView()}, errs: []error{nil}}
	h := newClientHandler(nil, update, nil, nil, getByID, nil)
	app := testutil.NewTestApp()
	app.Put("/clients/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Update)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPut, "/clients/"+testutil.TestClientID.String(), map[string]any{"name": "Updated"}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if !update.called || update.cmd.Name != "Updated" {
		t.Fatalf("update cmd: %+v", update.cmd)
	}
}

func TestClientHandler_Update_NotFound(t *testing.T) {
	update := &mockUpdateClientHandler{err: errors.New("client not found")}
	h := newClientHandler(nil, update, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Put("/clients/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Update)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPut, "/clients/"+testutil.TestClientID.String(), map[string]any{"name": "Updated"}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestClientHandler_Delete_Success(t *testing.T) {
	deleteH := &mockDeleteClientHandler{}
	h := newClientHandler(nil, nil, deleteH, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Delete("/clients/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Delete)

	resp, err := app.Test(mustJSONRequest(t, http.MethodDelete, "/clients/"+testutil.TestClientID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if !deleteH.called || deleteH.id != testutil.TestClientID {
		t.Fatalf("delete: %+v", deleteH)
	}
}

func TestClientHandler_Delete_InvalidID(t *testing.T) {
	h := newClientHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Delete("/clients/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Delete)

	resp, err := app.Test(mustJSONRequest(t, http.MethodDelete, "/clients/bad", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestClientHandler_Delete_HandlerError_Internal(t *testing.T) {
	deleteH := &mockDeleteClientHandler{err: errors.New("boom")}
	h := newClientHandler(nil, nil, deleteH, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Delete("/clients/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Delete)

	resp, err := app.Test(mustJSONRequest(t, http.MethodDelete, "/clients/"+testutil.TestClientID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestClientHandler_RemoveMember_Success(t *testing.T) {
	remove := &mockRemoveMemberHandler{}
	h := newClientHandler(nil, nil, nil, remove, nil, nil)
	app := testutil.NewTestApp()
	app.Delete("/clients/:id/members/:userId", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.RemoveMember)

	path := "/clients/" + testutil.TestClientID.String() + "/members/" + testutil.TestUserID.String()
	resp, err := app.Test(mustJSONRequest(t, http.MethodDelete, path, nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if !remove.called {
		t.Fatal("remove not called")
	}
}

func TestClientHandler_RemoveMember_NotFound(t *testing.T) {
	remove := &mockRemoveMemberHandler{err: errors.New("client not found")}
	h := newClientHandler(nil, nil, nil, remove, nil, nil)
	app := testutil.NewTestApp()
	app.Delete("/clients/:id/members/:userId", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.RemoveMember)

	path := "/clients/" + testutil.TestClientID.String() + "/members/" + testutil.TestUserID.String()
	resp, err := app.Test(mustJSONRequest(t, http.MethodDelete, path, nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestClientHandler_List_HandlerError_Internal(t *testing.T) {
	list := &mockListClientsByUserHandler{err: errors.New("db")}
	h := newClientHandler(nil, nil, nil, nil, nil, list)
	app := testutil.NewTestApp()
	app.Get("/clients", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.List)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/clients", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestClientHandler_GetByID_HandlerError_Internal(t *testing.T) {
	getByID := &mockGetClientByIDHandler{views: []*domainclient.ClientView{nil}, errs: []error{errors.New("db")}}
	h := newClientHandler(nil, nil, nil, nil, getByID, nil)
	app := testutil.NewTestApp()
	app.Get("/clients/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetByID)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/clients/"+testutil.TestClientID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestClientHandler_Update_Unauthorized(t *testing.T) {
	h := newClientHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Put("/clients/:id", h.Update)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPut, "/clients/"+testutil.TestClientID.String(), map[string]any{"name": "x"}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestClientHandler_Update_InvalidID(t *testing.T) {
	h := newClientHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Put("/clients/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Update)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPut, "/clients/bad", map[string]any{"name": "x"}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestClientHandler_Update_InvalidInput(t *testing.T) {
	update := &mockUpdateClientHandler{}
	h := newClientHandler(nil, update, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Put("/clients/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Update)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPut, "/clients/"+testutil.TestClientID.String(), map[string]any{"name": ""}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if update.called {
		t.Fatal("update must not be called")
	}
}

func TestClientHandler_Update_HandlerError_Internal(t *testing.T) {
	update := &mockUpdateClientHandler{err: errors.New("boom")}
	h := newClientHandler(nil, update, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Put("/clients/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Update)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPut, "/clients/"+testutil.TestClientID.String(), map[string]any{"name": "Updated"}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestClientHandler_Update_ReloadFailure(t *testing.T) {
	update := &mockUpdateClientHandler{}
	getByID := &mockGetClientByIDHandler{views: []*domainclient.ClientView{nil}, errs: []error{errors.New("reload")}}
	h := newClientHandler(nil, update, nil, nil, getByID, nil)
	app := testutil.NewTestApp()
	app.Put("/clients/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Update)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPut, "/clients/"+testutil.TestClientID.String(), map[string]any{"name": "Updated"}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestClientHandler_RemoveMember_InvalidClientID(t *testing.T) {
	h := newClientHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Delete("/clients/:id/members/:userId", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.RemoveMember)

	resp, err := app.Test(mustJSONRequest(t, http.MethodDelete, "/clients/bad/members/"+testutil.TestUserID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestClientHandler_RemoveMember_InvalidUserID(t *testing.T) {
	h := newClientHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Delete("/clients/:id/members/:userId", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.RemoveMember)

	resp, err := app.Test(mustJSONRequest(t, http.MethodDelete, "/clients/"+testutil.TestClientID.String()+"/members/bad", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestClientHandler_RemoveMember_HandlerError_Internal(t *testing.T) {
	remove := &mockRemoveMemberHandler{err: errors.New("boom")}
	h := newClientHandler(nil, nil, nil, remove, nil, nil)
	app := testutil.NewTestApp()
	app.Delete("/clients/:id/members/:userId", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.RemoveMember)

	path := "/clients/" + testutil.TestClientID.String() + "/members/" + testutil.TestUserID.String()
	resp, err := app.Test(mustJSONRequest(t, http.MethodDelete, path, nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}
