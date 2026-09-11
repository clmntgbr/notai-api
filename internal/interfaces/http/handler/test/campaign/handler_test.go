package campaigntest

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	campaigncmd "go-api/internal/application/command/campaign"
	querycampaign "go-api/internal/application/query/campaign"
	queryclient "go-api/internal/application/query/client"
	domaincampaign "go-api/internal/domain/campaign"
	domainclient "go-api/internal/domain/client"
	"go-api/internal/domain/paginate"
	"go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/testutil"

	"github.com/google/uuid"
)

var otherClientID = uuid.MustParse("01960000-0000-7000-8000-00000000000b")

type mockCreateCampaignHandler struct {
	called bool
	cmd    campaigncmd.CreateCampaignCommand
	result *domaincampaign.Campaign
	err    error
}

func (m *mockCreateCampaignHandler) Handle(
	_ context.Context,
	cmd campaigncmd.CreateCampaignCommand,
) (*domaincampaign.Campaign, error) {
	m.called = true
	m.cmd = cmd
	return m.result, m.err
}

type mockUpdateCampaignHandler struct {
	called bool
	cmd    campaigncmd.UpdateCampaignCommand
	err    error
}

func (m *mockUpdateCampaignHandler) Handle(_ context.Context, cmd campaigncmd.UpdateCampaignCommand) error {
	m.called = true
	m.cmd = cmd
	return m.err
}

type mockDeleteCampaignHandler struct {
	called bool
	cmd    campaigncmd.DeleteCampaignCommand
	err    error
}

func (m *mockDeleteCampaignHandler) Handle(_ context.Context, cmd campaigncmd.DeleteCampaignCommand) error {
	m.called = true
	m.cmd = cmd
	return m.err
}

type mockGetCampaignByIDHandler struct {
	calls int
	views []*domaincampaign.CampaignView
	errs  []error
}

func (m *mockGetCampaignByIDHandler) Handle(
	_ context.Context,
	_ querycampaign.GetCampaignByIDQuery,
) (*domaincampaign.CampaignView, error) {
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

type mockListCampaignsByClientHandler struct {
	called bool
	query  querycampaign.ListCampaignsByClientQuery
	views  []domaincampaign.CampaignView
	total  int64
	err    error
}

func (m *mockListCampaignsByClientHandler) Handle(
	_ context.Context,
	q querycampaign.ListCampaignsByClientQuery,
) ([]domaincampaign.CampaignView, int64, error) {
	m.called = true
	m.query = q
	return m.views, m.total, m.err
}

type mockGetClientByIDHandler struct {
	view *domainclient.ClientView
	err  error
}

func (m *mockGetClientByIDHandler) Handle(
	_ context.Context,
	_ queryclient.GetClientByIDQuery,
) (*domainclient.ClientView, error) {
	return m.view, m.err
}

func newCampaignHandler(
	create *mockCreateCampaignHandler,
	update *mockUpdateCampaignHandler,
	deleteH *mockDeleteCampaignHandler,
	getByID *mockGetCampaignByIDHandler,
	list *mockListCampaignsByClientHandler,
	getClient *mockGetClientByIDHandler,
) *handler.CampaignHandler {
	return newCampaignHandlerWithExtras(create, update, deleteH, getByID, list, getClient, nil, nil, nil)
}

func newCampaignHandlerWithExtras(
	create *mockCreateCampaignHandler,
	update *mockUpdateCampaignHandler,
	deleteH *mockDeleteCampaignHandler,
	getByID *mockGetCampaignByIDHandler,
	list *mockListCampaignsByClientHandler,
	getClient *mockGetClientByIDHandler,
	presign *mockPresignBackgroundHandler,
	clear *mockClearBackgroundHandler,
	storage campaignThumbnailStorage,
) *handler.CampaignHandler {
	if create == nil {
		create = &mockCreateCampaignHandler{}
	}
	if update == nil {
		update = &mockUpdateCampaignHandler{}
	}
	if deleteH == nil {
		deleteH = &mockDeleteCampaignHandler{}
	}
	if getByID == nil {
		getByID = &mockGetCampaignByIDHandler{}
	}
	if list == nil {
		list = &mockListCampaignsByClientHandler{}
	}
	if getClient == nil {
		getClient = &mockGetClientByIDHandler{}
	}
	if presign == nil {
		presign = &mockPresignBackgroundHandler{}
	}
	if clear == nil {
		clear = &mockClearBackgroundHandler{}
	}
	if storage == nil {
		storage = &mockCampaignStorage{}
	}
	return handler.NewCampaignHandler(
		create, update, deleteH, getByID, list, getClient, presign, clear, storage,
	)
}

type campaignThumbnailStorage interface {
	GetThumbnail(ctx context.Context, key string) (io.ReadCloser, error)
}

type mockPresignBackgroundHandler struct {
	called bool
	cmd    campaigncmd.PresignBackgroundCommand
	result *campaigncmd.PresignBackgroundResult
	err    error
}

func (m *mockPresignBackgroundHandler) Handle(
	_ context.Context,
	cmd campaigncmd.PresignBackgroundCommand,
) (*campaigncmd.PresignBackgroundResult, error) {
	m.called = true
	m.cmd = cmd
	return m.result, m.err
}

type mockClearBackgroundHandler struct {
	called bool
	cmd    campaigncmd.ClearBackgroundCommand
	err    error
}

func (m *mockClearBackgroundHandler) Handle(
	_ context.Context,
	cmd campaigncmd.ClearBackgroundCommand,
) error {
	m.called = true
	m.cmd = cmd
	return m.err
}

type mockCampaignStorage struct {
	called bool
	key    string
	body   []byte
	err    error
}

func (m *mockCampaignStorage) GetThumbnail(_ context.Context, key string) (io.ReadCloser, error) {
	m.called = true
	m.key = key
	if m.err != nil {
		return nil, m.err
	}
	return io.NopCloser(bytes.NewReader(m.body)), nil
}

func sampleCampaignEntity() *domaincampaign.Campaign {
	return &domaincampaign.Campaign{
		ID:        testutil.TestCampaignID,
		ClientID:  testutil.TestClientID,
		Name:      "Spring Launch",
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func sampleCampaignView() *domaincampaign.CampaignView {
	e := sampleCampaignEntity()
	return &domaincampaign.CampaignView{
		ID:                      e.ID,
		ClientID:                e.ClientID,
		Name:                    e.Name,
		CreatedAt:               e.CreatedAt,
		UpdatedAt:               e.UpdatedAt,
		ContentFailedCount:      1,
		ContentHumanCount:       2,
		ContentAIGeneratedCount: 3,
		ContentUncertainCount:   4,
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

func TestCampaignHandler_Create_Success(t *testing.T) {
	create := &mockCreateCampaignHandler{result: sampleCampaignEntity()}
	h := newCampaignHandler(create, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post("/campaigns", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Create)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost, "/campaigns", map[string]any{"name": "Spring Launch"}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if !create.called || create.cmd.ClientID != testutil.TestClientID || create.cmd.Name != "Spring Launch" {
		t.Fatalf("create cmd: %+v", create.cmd)
	}
}

func TestCampaignHandler_Create_InvalidSchedule(t *testing.T) {
	create := &mockCreateCampaignHandler{err: domaincampaign.ErrInvalidSchedule}
	h := newCampaignHandler(create, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post("/campaigns", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Create)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost, "/campaigns", map[string]any{
		"name":    "Spring Launch",
		"startAt": "2026-06-01T00:00:00Z",
		"endAt":   "2026-01-01T00:00:00Z",
	}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_Create_Unauthorized(t *testing.T) {
	h := newCampaignHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post("/campaigns", h.Create)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost, "/campaigns", map[string]any{"name": "x"}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_Create_MissingActiveClient(t *testing.T) {
	create := &mockCreateCampaignHandler{}
	h := newCampaignHandler(create, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post("/campaigns", testutil.WithUserWithoutClient(testutil.TestUserID), h.Create)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost, "/campaigns", map[string]any{"name": "x"}))
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

func TestCampaignHandler_Create_InvalidInput(t *testing.T) {
	create := &mockCreateCampaignHandler{}
	h := newCampaignHandler(create, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post("/campaigns", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Create)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost, "/campaigns", map[string]any{"name": ""}))
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

func TestCampaignHandler_Create_HandlerError_Internal(t *testing.T) {
	create := &mockCreateCampaignHandler{err: errors.New("boom")}
	h := newCampaignHandler(create, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post("/campaigns", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Create)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost, "/campaigns", map[string]any{"name": "Spring Launch"}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_List_Success(t *testing.T) {
	view := sampleCampaignView()
	list := &mockListCampaignsByClientHandler{views: []domaincampaign.CampaignView{*view}, total: 1}
	h := newCampaignHandler(nil, nil, nil, nil, list, nil)
	app := testutil.NewTestApp()
	app.Get("/campaigns", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.List)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/campaigns", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if !list.called || list.query.ClientID != testutil.TestClientID {
		t.Fatalf("list query: %+v", list.query)
	}
	if list.query.Query.OrderBy != paginate.OrderByDesc {
		t.Fatalf("order: %s", list.query.Query.OrderBy)
	}
	body := testutil.DecodeJSONMap(t, resp)
	members, ok := body["members"].([]any)
	if !ok || len(members) != 1 {
		t.Fatalf("members: %#v", body["members"])
	}
	member, _ := members[0].(map[string]any)
	counts, ok := member["contentCounts"].(map[string]any)
	if !ok {
		t.Fatalf("contentCounts: %#v", member["contentCounts"])
	}
	if counts["failed"] != float64(1) || counts["human"] != float64(2) {
		t.Fatalf("contentCounts: %#v", counts)
	}
}

func TestCampaignHandler_List_MissingActiveClient(t *testing.T) {
	h := newCampaignHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/campaigns", testutil.WithUserWithoutClient(testutil.TestUserID), h.List)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/campaigns", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_GetByID_Success(t *testing.T) {
	getByID := &mockGetCampaignByIDHandler{views: []*domaincampaign.CampaignView{sampleCampaignView()}, errs: []error{nil}}
	h := newCampaignHandler(nil, nil, nil, getByID, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/campaigns/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetByID)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/campaigns/"+testutil.TestCampaignID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	body := testutil.DecodeJSONMap(t, resp)
	counts, ok := body["contentCounts"].(map[string]any)
	if !ok {
		t.Fatalf("contentCounts: %#v", body["contentCounts"])
	}
	if counts["failed"] != float64(1) || counts["human"] != float64(2) ||
		counts["aiGenerated"] != float64(3) || counts["uncertain"] != float64(4) {
		t.Fatalf("contentCounts: %#v", counts)
	}
}

func TestCampaignHandler_GetByID_WrongClient(t *testing.T) {
	view := sampleCampaignView()
	view.ClientID = otherClientID
	getByID := &mockGetCampaignByIDHandler{views: []*domaincampaign.CampaignView{view}, errs: []error{nil}}
	getClient := &mockGetClientByIDHandler{
		view: &domainclient.ClientView{
			ID:        otherClientID,
			Name:      "Other",
			MemberIDs: []uuid.UUID{testutil.TestUserID},
		},
	}
	h := newCampaignHandler(nil, nil, nil, getByID, nil, getClient)
	app := testutil.NewTestApp()
	app.Get("/campaigns/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetByID)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/campaigns/"+testutil.TestCampaignID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	body := testutil.DecodeJSONMap(t, resp)
	if body["code"] != "WRONG_CLIENT" {
		t.Fatalf("code: %+v", body)
	}
}

func TestCampaignHandler_GetByID_WrongClient_NotMember(t *testing.T) {
	view := sampleCampaignView()
	view.ClientID = otherClientID
	getByID := &mockGetCampaignByIDHandler{views: []*domaincampaign.CampaignView{view}, errs: []error{nil}}
	getClient := &mockGetClientByIDHandler{
		view: &domainclient.ClientView{
			ID:        otherClientID,
			Name:      "Other",
			MemberIDs: []uuid.UUID{uuid.MustParse("01960000-0000-7000-8000-000000000099")},
		},
	}
	h := newCampaignHandler(nil, nil, nil, getByID, nil, getClient)
	app := testutil.NewTestApp()
	app.Get("/campaigns/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetByID)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/campaigns/"+testutil.TestCampaignID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_GetByID_NotFound(t *testing.T) {
	getByID := &mockGetCampaignByIDHandler{views: []*domaincampaign.CampaignView{nil}, errs: []error{errors.New("campaign not found")}}
	h := newCampaignHandler(nil, nil, nil, getByID, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/campaigns/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetByID)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/campaigns/"+testutil.TestCampaignID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_GetByID_InvalidID(t *testing.T) {
	h := newCampaignHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/campaigns/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetByID)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/campaigns/bad", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_Update_Success(t *testing.T) {
	update := &mockUpdateCampaignHandler{}
	getByID := &mockGetCampaignByIDHandler{
		views: []*domaincampaign.CampaignView{sampleCampaignView(), sampleCampaignView()},
		errs:  []error{nil, nil},
	}
	h := newCampaignHandler(nil, update, nil, getByID, nil, nil)
	app := testutil.NewTestApp()
	app.Put("/campaigns/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Update)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPut, "/campaigns/"+testutil.TestCampaignID.String(), map[string]any{"name": "Updated"}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if !update.called || update.cmd.Name != "Updated" {
		t.Fatalf("update: %+v", update.cmd)
	}
}

func TestCampaignHandler_Update_InvalidSchedule(t *testing.T) {
	update := &mockUpdateCampaignHandler{err: domaincampaign.ErrInvalidSchedule}
	getByID := &mockGetCampaignByIDHandler{
		views: []*domaincampaign.CampaignView{sampleCampaignView()},
		errs:  []error{nil},
	}
	h := newCampaignHandler(nil, update, nil, getByID, nil, nil)
	app := testutil.NewTestApp()
	app.Put("/campaigns/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Update)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPut, "/campaigns/"+testutil.TestCampaignID.String(), map[string]any{
		"name":    "Updated",
		"startAt": "2026-06-01T00:00:00Z",
		"endAt":   "2026-01-01T00:00:00Z",
	}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_Update_WrongClient(t *testing.T) {
	view := sampleCampaignView()
	view.ClientID = otherClientID
	getByID := &mockGetCampaignByIDHandler{views: []*domaincampaign.CampaignView{view}, errs: []error{nil}}
	update := &mockUpdateCampaignHandler{}
	h := newCampaignHandler(nil, update, nil, getByID, nil, nil)
	app := testutil.NewTestApp()
	app.Put("/campaigns/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Update)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPut, "/campaigns/"+testutil.TestCampaignID.String(), map[string]any{"name": "Updated"}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if update.called {
		t.Fatal("update must not be called")
	}
}

func TestCampaignHandler_Delete_Success(t *testing.T) {
	deleteH := &mockDeleteCampaignHandler{}
	getByID := &mockGetCampaignByIDHandler{views: []*domaincampaign.CampaignView{sampleCampaignView()}, errs: []error{nil}}
	h := newCampaignHandler(nil, nil, deleteH, getByID, nil, nil)
	app := testutil.NewTestApp()
	app.Delete("/campaigns/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Delete)

	resp, err := app.Test(mustJSONRequest(t, http.MethodDelete, "/campaigns/"+testutil.TestCampaignID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if !deleteH.called {
		t.Fatal("delete not called")
	}
}

func TestCampaignHandler_Delete_WrongClient(t *testing.T) {
	view := sampleCampaignView()
	view.ClientID = otherClientID
	getByID := &mockGetCampaignByIDHandler{views: []*domaincampaign.CampaignView{view}, errs: []error{nil}}
	deleteH := &mockDeleteCampaignHandler{}
	h := newCampaignHandler(nil, nil, deleteH, getByID, nil, nil)
	app := testutil.NewTestApp()
	app.Delete("/campaigns/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Delete)

	resp, err := app.Test(mustJSONRequest(t, http.MethodDelete, "/campaigns/"+testutil.TestCampaignID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if deleteH.called {
		t.Fatal("delete must not be called")
	}
}

func TestCampaignHandler_Delete_MissingActiveClient(t *testing.T) {
	h := newCampaignHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Delete("/campaigns/:id", testutil.WithUserWithoutClient(testutil.TestUserID), h.Delete)

	resp, err := app.Test(mustJSONRequest(t, http.MethodDelete, "/campaigns/"+testutil.TestCampaignID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_GetByID_Unauthorized(t *testing.T) {
	h := newCampaignHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/campaigns/:id", h.GetByID)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/campaigns/"+testutil.TestCampaignID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_GetByID_MissingActiveClient(t *testing.T) {
	h := newCampaignHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/campaigns/:id", testutil.WithUserWithoutClient(testutil.TestUserID), h.GetByID)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/campaigns/"+testutil.TestCampaignID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_GetByID_HandlerError_Internal(t *testing.T) {
	getByID := &mockGetCampaignByIDHandler{views: []*domaincampaign.CampaignView{nil}, errs: []error{errors.New("db")}}
	h := newCampaignHandler(nil, nil, nil, getByID, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/campaigns/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetByID)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/campaigns/"+testutil.TestCampaignID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_GetByID_OwnerClientMissing(t *testing.T) {
	view := sampleCampaignView()
	view.ClientID = otherClientID
	getByID := &mockGetCampaignByIDHandler{views: []*domaincampaign.CampaignView{view}, errs: []error{nil}}
	getClient := &mockGetClientByIDHandler{err: errors.New("client not found")}
	h := newCampaignHandler(nil, nil, nil, getByID, nil, getClient)
	app := testutil.NewTestApp()
	app.Get("/campaigns/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetByID)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/campaigns/"+testutil.TestCampaignID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_List_HandlerError_Internal(t *testing.T) {
	list := &mockListCampaignsByClientHandler{err: errors.New("db")}
	h := newCampaignHandler(nil, nil, nil, nil, list, nil)
	app := testutil.NewTestApp()
	app.Get("/campaigns", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.List)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/campaigns", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_Update_Unauthorized(t *testing.T) {
	h := newCampaignHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Put("/campaigns/:id", h.Update)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPut, "/campaigns/"+testutil.TestCampaignID.String(), map[string]any{"name": "x"}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_Update_MissingActiveClient(t *testing.T) {
	h := newCampaignHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Put("/campaigns/:id", testutil.WithUserWithoutClient(testutil.TestUserID), h.Update)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPut, "/campaigns/"+testutil.TestCampaignID.String(), map[string]any{"name": "x"}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_Update_InvalidID(t *testing.T) {
	h := newCampaignHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Put("/campaigns/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Update)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPut, "/campaigns/bad", map[string]any{"name": "x"}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_Update_NotFound(t *testing.T) {
	getByID := &mockGetCampaignByIDHandler{views: []*domaincampaign.CampaignView{nil}, errs: []error{errors.New("campaign not found")}}
	h := newCampaignHandler(nil, nil, nil, getByID, nil, nil)
	app := testutil.NewTestApp()
	app.Put("/campaigns/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Update)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPut, "/campaigns/"+testutil.TestCampaignID.String(), map[string]any{"name": "x"}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_Update_GetExistingInternal(t *testing.T) {
	getByID := &mockGetCampaignByIDHandler{views: []*domaincampaign.CampaignView{nil}, errs: []error{errors.New("db")}}
	h := newCampaignHandler(nil, nil, nil, getByID, nil, nil)
	app := testutil.NewTestApp()
	app.Put("/campaigns/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Update)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPut, "/campaigns/"+testutil.TestCampaignID.String(), map[string]any{"name": "x"}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_Update_InvalidInput(t *testing.T) {
	getByID := &mockGetCampaignByIDHandler{views: []*domaincampaign.CampaignView{sampleCampaignView()}, errs: []error{nil}}
	update := &mockUpdateCampaignHandler{}
	h := newCampaignHandler(nil, update, nil, getByID, nil, nil)
	app := testutil.NewTestApp()
	app.Put("/campaigns/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Update)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPut, "/campaigns/"+testutil.TestCampaignID.String(), map[string]any{"name": ""}))
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

func TestCampaignHandler_Update_HandlerNotFound(t *testing.T) {
	getByID := &mockGetCampaignByIDHandler{views: []*domaincampaign.CampaignView{sampleCampaignView()}, errs: []error{nil}}
	update := &mockUpdateCampaignHandler{err: errors.New("campaign not found")}
	h := newCampaignHandler(nil, update, nil, getByID, nil, nil)
	app := testutil.NewTestApp()
	app.Put("/campaigns/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Update)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPut, "/campaigns/"+testutil.TestCampaignID.String(), map[string]any{"name": "Updated"}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_Update_DefaultProtected(t *testing.T) {
	getByID := &mockGetCampaignByIDHandler{views: []*domaincampaign.CampaignView{sampleCampaignView()}, errs: []error{nil}}
	update := &mockUpdateCampaignHandler{err: domaincampaign.ErrDefaultCampaignProtected}
	h := newCampaignHandler(nil, update, nil, getByID, nil, nil)
	app := testutil.NewTestApp()
	app.Put("/campaigns/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Update)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPut, "/campaigns/"+testutil.TestCampaignID.String(), map[string]any{"name": "Nope"}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	body := testutil.DecodeJSONMap(t, resp)
	if body["code"] != "DEFAULT_CAMPAIGN_PROTECTED" {
		t.Fatalf("code: got %#v", body["code"])
	}
}

func TestCampaignHandler_Delete_DefaultProtected(t *testing.T) {
	getByID := &mockGetCampaignByIDHandler{views: []*domaincampaign.CampaignView{sampleCampaignView()}, errs: []error{nil}}
	deleteH := &mockDeleteCampaignHandler{err: domaincampaign.ErrDefaultCampaignProtected}
	h := newCampaignHandler(nil, nil, deleteH, getByID, nil, nil)
	app := testutil.NewTestApp()
	app.Delete("/campaigns/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Delete)

	resp, err := app.Test(mustJSONRequest(t, http.MethodDelete, "/campaigns/"+testutil.TestCampaignID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_Update_HandlerError_Internal(t *testing.T) {
	getByID := &mockGetCampaignByIDHandler{views: []*domaincampaign.CampaignView{sampleCampaignView()}, errs: []error{nil}}
	update := &mockUpdateCampaignHandler{err: errors.New("boom")}
	h := newCampaignHandler(nil, update, nil, getByID, nil, nil)
	app := testutil.NewTestApp()
	app.Put("/campaigns/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Update)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPut, "/campaigns/"+testutil.TestCampaignID.String(), map[string]any{"name": "Updated"}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_Update_ReloadFailure(t *testing.T) {
	getByID := &mockGetCampaignByIDHandler{
		views: []*domaincampaign.CampaignView{sampleCampaignView(), nil},
		errs:  []error{nil, errors.New("reload")},
	}
	update := &mockUpdateCampaignHandler{}
	h := newCampaignHandler(nil, update, nil, getByID, nil, nil)
	app := testutil.NewTestApp()
	app.Put("/campaigns/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Update)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPut, "/campaigns/"+testutil.TestCampaignID.String(), map[string]any{"name": "Updated"}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_Delete_Unauthorized(t *testing.T) {
	h := newCampaignHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Delete("/campaigns/:id", h.Delete)

	resp, err := app.Test(mustJSONRequest(t, http.MethodDelete, "/campaigns/"+testutil.TestCampaignID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_Delete_InvalidID(t *testing.T) {
	h := newCampaignHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Delete("/campaigns/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Delete)

	resp, err := app.Test(mustJSONRequest(t, http.MethodDelete, "/campaigns/bad", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_Delete_NotFound(t *testing.T) {
	getByID := &mockGetCampaignByIDHandler{views: []*domaincampaign.CampaignView{nil}, errs: []error{errors.New("campaign not found")}}
	h := newCampaignHandler(nil, nil, nil, getByID, nil, nil)
	app := testutil.NewTestApp()
	app.Delete("/campaigns/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Delete)

	resp, err := app.Test(mustJSONRequest(t, http.MethodDelete, "/campaigns/"+testutil.TestCampaignID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_Delete_GetInternal(t *testing.T) {
	getByID := &mockGetCampaignByIDHandler{views: []*domaincampaign.CampaignView{nil}, errs: []error{errors.New("db")}}
	h := newCampaignHandler(nil, nil, nil, getByID, nil, nil)
	app := testutil.NewTestApp()
	app.Delete("/campaigns/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Delete)

	resp, err := app.Test(mustJSONRequest(t, http.MethodDelete, "/campaigns/"+testutil.TestCampaignID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_Delete_HandlerError_Internal(t *testing.T) {
	getByID := &mockGetCampaignByIDHandler{views: []*domaincampaign.CampaignView{sampleCampaignView()}, errs: []error{nil}}
	deleteH := &mockDeleteCampaignHandler{err: errors.New("boom")}
	h := newCampaignHandler(nil, nil, deleteH, getByID, nil, nil)
	app := testutil.NewTestApp()
	app.Delete("/campaigns/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Delete)

	resp, err := app.Test(mustJSONRequest(t, http.MethodDelete, "/campaigns/"+testutil.TestCampaignID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}
