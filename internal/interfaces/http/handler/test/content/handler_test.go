package contenttest

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	querycontent "go-api/internal/application/query/content"
	domaincampaign "go-api/internal/domain/campaign"
	domaincontent "go-api/internal/domain/content"
	"go-api/internal/domain/paginate"
	"go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/testutil"

	"github.com/google/uuid"
)

type mockGetContentByIDHandler struct {
	view *domaincontent.ContentView
	err  error
}

func (m *mockGetContentByIDHandler) Handle(
	_ context.Context,
	_ querycontent.GetContentByIDQuery,
) (*domaincontent.ContentView, error) {
	return m.view, m.err
}

type mockListContentsByCampaignHandler struct {
	called bool
	query  querycontent.ListContentsByCampaignQuery
	result *querycontent.ListContentsByCampaignResult
	err    error
}

func (m *mockListContentsByCampaignHandler) Handle(
	_ context.Context,
	q querycontent.ListContentsByCampaignQuery,
) (*querycontent.ListContentsByCampaignResult, error) {
	m.called = true
	m.query = q
	if m.err != nil {
		return nil, m.err
	}
	if m.result != nil {
		return m.result, nil
	}
	return &querycontent.ListContentsByCampaignResult{}, nil
}

type mockContentStatsByClientHandler struct {
	called bool
	query  querycontent.GetContentStatsByClientQuery
	stats  *domaincontent.ContentStats
	err    error
}

func (m *mockContentStatsByClientHandler) Handle(
	_ context.Context,
	q querycontent.GetContentStatsByClientQuery,
) (*domaincontent.ContentStats, error) {
	m.called = true
	m.query = q
	return m.stats, m.err
}

type mockContentStorage struct {
	called bool
	key    string
	body   []byte
	err    error
}

func (m *mockContentStorage) GetThumbnail(_ context.Context, key string) (io.ReadCloser, error) {
	m.called = true
	m.key = key
	if m.err != nil {
		return nil, m.err
	}
	return io.NopCloser(bytes.NewReader(m.body)), nil
}

func newContentHandler(
	getByID *mockGetContentByIDHandler,
	list *mockListContentsByCampaignHandler,
	stats *mockContentStatsByClientHandler,
	storage *mockContentStorage,
) *handler.ContentHandler {
	if getByID == nil {
		getByID = &mockGetContentByIDHandler{}
	}
	if list == nil {
		list = &mockListContentsByCampaignHandler{}
	}
	if stats == nil {
		stats = &mockContentStatsByClientHandler{}
	}
	if storage == nil {
		storage = &mockContentStorage{}
	}
	return handler.NewContentHandler(getByID, list, stats, storage)
}

func sampleContentView() *domaincontent.ContentView {
	size := int64(1024)
	thumb := "clients/x/campaigns/y/contents/thumbnails/z.jpg"
	return &domaincontent.ContentView{
		ID:           uuid.MustParse("01960000-0000-7000-8000-0000000000c1"),
		MediaID:      uuid.MustParse("01960000-0000-7000-8000-0000000000a1"),
		CampaignID:   testutil.TestCampaignID,
		ClientID:     testutil.TestClientID,
		Filename:     "shot.png",
		ContentType:  "image/png",
		ObjectKey:    "clients/x/campaigns/y/contents/z.png",
		ThumbnailKey: &thumb,
		SizeBytes:    &size,
		Status:       domaincontent.StatusUploaded,
		CreatedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
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


func TestContentHandler_GetThumbnail_Success(t *testing.T) {
	thumbKey := "clients/x/campaigns/y/contents/thumbnails/z.jpg"
	getByID := &mockGetContentByIDHandler{
		view: &domaincontent.ContentView{
			ID:           testutil.TestCampaignID,
			ClientID:     testutil.TestClientID,
			CampaignID:   testutil.TestCampaignID,
			ThumbnailKey: &thumbKey,
			Status:       domaincontent.StatusUploaded,
			UpdatedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	storage := &mockContentStorage{body: []byte{0xff, 0xd8, 0xff}}
	h := newContentHandler(getByID, nil, nil, storage)
	app := testutil.NewTestApp()
	app.Get(
		"/contents/:id/thumbnail",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.GetThumbnail,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/contents/"+testutil.TestCampaignID.String()+"/thumbnail",
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if !storage.called || storage.key != thumbKey {
		t.Fatalf("storage: %+v", storage)
	}
}

func TestContentHandler_GetThumbnail_Missing(t *testing.T) {
	getByID := &mockGetContentByIDHandler{
		view: &domaincontent.ContentView{
			ID:       testutil.TestCampaignID,
			ClientID: testutil.TestClientID,
			Status:   domaincontent.StatusPendingUpload,
		},
	}
	h := newContentHandler(getByID, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get(
		"/contents/:id/thumbnail",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.GetThumbnail,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/contents/"+testutil.TestCampaignID.String()+"/thumbnail",
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestContentHandler_GetThumbnail_WrongClient(t *testing.T) {
	thumbKey := "thumb.jpg"
	getByID := &mockGetContentByIDHandler{
		view: &domaincontent.ContentView{
			ID:           testutil.TestCampaignID,
			ClientID:     uuid.MustParse("01960000-0000-7000-8000-00000000000b"),
			ThumbnailKey: &thumbKey,
			Status:       domaincontent.StatusUploaded,
		},
	}
	h := newContentHandler(getByID, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get(
		"/contents/:id/thumbnail",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.GetThumbnail,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/contents/"+testutil.TestCampaignID.String()+"/thumbnail",
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestContentHandler_GetThumbnail_Unauthorized(t *testing.T) {
	h := newContentHandler(nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/contents/:id/thumbnail", h.GetThumbnail)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/contents/"+testutil.TestCampaignID.String()+"/thumbnail",
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestContentHandler_GetThumbnail_NotFound(t *testing.T) {
	getByID := &mockGetContentByIDHandler{err: errors.New("content not found")}
	h := newContentHandler(getByID, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get(
		"/contents/:id/thumbnail",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.GetThumbnail,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/contents/"+testutil.TestCampaignID.String()+"/thumbnail",
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}


func TestContentHandler_List_Success(t *testing.T) {
	campaignID := testutil.TestCampaignID
	list := &mockListContentsByCampaignHandler{
		result: &querycontent.ListContentsByCampaignResult{
			Views: []domaincontent.ContentView{*sampleContentView()},
			Total: 1,
			Campaigns: map[uuid.UUID]*domaincampaign.CampaignView{
				campaignID: {
					ID:        campaignID,
					ClientID:  testutil.TestClientID,
					Name:      "Spring Launch",
					IsDefault: false,
				},
			},
		},
	}
	h := newContentHandler(nil, list, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/contents", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.List)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/contents?campaignId="+campaignID.String(),
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if !list.called || list.query.CampaignID != campaignID || list.query.ClientID != testutil.TestClientID {
		t.Fatalf("list query: %+v", list.query)
	}
	if list.query.Query.OrderBy != paginate.OrderByDesc {
		t.Fatalf("order: %s", list.query.Query.OrderBy)
	}
	body := testutil.DecodeJSONMap(t, resp)
	members, _ := body["members"].([]any)
	if len(members) != 1 {
		t.Fatalf("members: %#v", body["members"])
	}
	member, _ := members[0].(map[string]any)
	campaign, ok := member["campaign"].(map[string]any)
	if !ok || campaign["name"] != "Spring Launch" {
		t.Fatalf("campaign: %#v", member["campaign"])
	}
}

func TestContentHandler_List_DefaultCampaign_OmitsCampaign(t *testing.T) {
	campaignID := testutil.TestCampaignID
	list := &mockListContentsByCampaignHandler{
		result: &querycontent.ListContentsByCampaignResult{
			Views: []domaincontent.ContentView{*sampleContentView()},
			Total: 1,
			Campaigns: map[uuid.UUID]*domaincampaign.CampaignView{
				campaignID: {
					ID:        campaignID,
					ClientID:  testutil.TestClientID,
					Name:      "Default",
					IsDefault: true,
				},
			},
		},
	}
	h := newContentHandler(nil, list, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/contents", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.List)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/contents", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	body := testutil.DecodeJSONMap(t, resp)
	members, _ := body["members"].([]any)
	member, _ := members[0].(map[string]any)
	if _, ok := member["campaign"]; ok {
		t.Fatalf("expected no campaign for default, got %#v", member["campaign"])
	}
}

func TestContentHandler_List_EmptyCampaignID(t *testing.T) {
	list := &mockListContentsByCampaignHandler{
		result: &querycontent.ListContentsByCampaignResult{Views: []domaincontent.ContentView{}, Total: 0},
	}
	h := newContentHandler(nil, list, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/contents", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.List)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/contents", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if !list.called || list.query.CampaignID != uuid.Nil {
		t.Fatalf("expected nil campaign id (all campaigns), got %+v", list.query.CampaignID)
	}
}

func TestContentHandler_List_Unauthorized(t *testing.T) {
	h := newContentHandler(nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/contents", h.List)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/contents", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestContentHandler_List_MissingClient(t *testing.T) {
	h := newContentHandler(nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/contents", testutil.WithUserWithoutClient(testutil.TestUserID), h.List)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/contents", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestContentHandler_List_InvalidCampaignID(t *testing.T) {
	h := newContentHandler(nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/contents", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.List)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/contents?campaignId=bad", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestContentHandler_List_CampaignNotFound(t *testing.T) {
	list := &mockListContentsByCampaignHandler{err: errors.New("campaign not found")}
	h := newContentHandler(nil, list, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/contents", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.List)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/contents", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestContentHandler_List_Internal(t *testing.T) {
	list := &mockListContentsByCampaignHandler{err: errors.New("boom")}
	h := newContentHandler(nil, list, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/contents", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.List)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/contents", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestContentHandler_GetByID_Success(t *testing.T) {
	getByID := &mockGetContentByIDHandler{view: sampleContentView()}
	h := newContentHandler(getByID, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/contents/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetByID)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/contents/"+sampleContentView().ID.String(),
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	body := testutil.DecodeJSONMap(t, resp)
	if body["filename"] != "shot.png" {
		t.Fatalf("body: %#v", body)
	}
}

func TestContentHandler_GetByID_Unauthorized(t *testing.T) {
	h := newContentHandler(nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/contents/:id", h.GetByID)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/contents/"+sampleContentView().ID.String(),
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestContentHandler_GetByID_MissingClient(t *testing.T) {
	h := newContentHandler(nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/contents/:id", testutil.WithUserWithoutClient(testutil.TestUserID), h.GetByID)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/contents/"+sampleContentView().ID.String(),
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestContentHandler_GetByID_InvalidID(t *testing.T) {
	h := newContentHandler(nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/contents/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetByID)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/contents/bad", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestContentHandler_GetByID_NotFound(t *testing.T) {
	getByID := &mockGetContentByIDHandler{err: errors.New("content not found")}
	h := newContentHandler(getByID, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/contents/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetByID)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/contents/"+sampleContentView().ID.String(),
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestContentHandler_GetByID_WrongClient(t *testing.T) {
	view := sampleContentView()
	view.ClientID = uuid.MustParse("01960000-0000-7000-8000-00000000000b")
	getByID := &mockGetContentByIDHandler{view: view}
	h := newContentHandler(getByID, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/contents/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetByID)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/contents/"+view.ID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestContentHandler_GetByID_Internal(t *testing.T) {
	getByID := &mockGetContentByIDHandler{err: errors.New("boom")}
	h := newContentHandler(getByID, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/contents/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetByID)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/contents/"+sampleContentView().ID.String(),
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestContentHandler_Stats_Success(t *testing.T) {
	stats := &mockContentStatsByClientHandler{
		stats: &domaincontent.ContentStats{
			PendingUpload: 5,
			Uploaded:      6,
			Analyzing:     7,
			Analyzed:      8,
			Failed:        1,
			Human:         2,
			AIGenerated:   3,
			Uncertain:     4,
			MonthlyControls: []domaincontent.ContentMonthlyStats{
				{Month: "2026-04", Analyzed: 1, Human: 1},
				{Month: "2026-05"},
				{Month: "2026-06", Analyzed: 2, AIGenerated: 2},
				{Month: "2026-07"},
				{Month: "2026-08"},
				{Month: "2026-09", Analyzed: 5, Human: 1, AIGenerated: 1, Uncertain: 3},
			},
		},
	}
	h := newContentHandler(nil, nil, stats, nil)
	app := testutil.NewTestApp()
	app.Get("/contents/stats", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Stats)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/contents/stats", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if !stats.called || stats.query.ClientID != testutil.TestClientID {
		t.Fatalf("stats query: %+v", stats.query)
	}
	body := testutil.DecodeJSONMap(t, resp)
	if body["failed"] != float64(1) || body["human"] != float64(2) ||
		body["aiGenerated"] != float64(3) || body["uncertain"] != float64(4) ||
		body["pendingUpload"] != float64(5) || body["uploaded"] != float64(6) ||
		body["analyzing"] != float64(7) || body["analyzed"] != float64(8) {
		t.Fatalf("body: %#v", body)
	}
	monthly, ok := body["monthlyControls"].([]any)
	if !ok || len(monthly) != 6 {
		t.Fatalf("monthlyControls: %#v", body["monthlyControls"])
	}
	first, ok := monthly[0].(map[string]any)
	if !ok || first["month"] != "2026-04" || first["analyzed"] != float64(1) {
		t.Fatalf("first month: %#v", monthly[0])
	}
	last, ok := monthly[5].(map[string]any)
	if !ok || last["month"] != "2026-09" || last["uncertain"] != float64(3) {
		t.Fatalf("last month: %#v", monthly[5])
	}
}

func TestContentHandler_Stats_Unauthorized(t *testing.T) {
	h := newContentHandler(nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/contents/stats", h.Stats)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/contents/stats", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestContentHandler_Stats_MissingClient(t *testing.T) {
	h := newContentHandler(nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/contents/stats", testutil.WithUserWithoutClient(testutil.TestUserID), h.Stats)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/contents/stats", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestContentHandler_Stats_Internal(t *testing.T) {
	stats := &mockContentStatsByClientHandler{err: errors.New("boom")}
	h := newContentHandler(nil, nil, stats, nil)
	app := testutil.NewTestApp()
	app.Get("/contents/stats", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Stats)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/contents/stats", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}
