package contenttest

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	contentcmd "go-api/internal/application/command/content"
	querycontent "go-api/internal/application/query/content"
	domaincampaign "go-api/internal/domain/campaign"
	domaincontent "go-api/internal/domain/content"
	"go-api/internal/domain/paginate"
	"go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/testutil"

	"github.com/google/uuid"
)

type mockPresignContentsHandler struct {
	called bool
	cmd    contentcmd.PresignContentsCommand
	result *contentcmd.PresignContentsResult
	err    error
}

func (m *mockPresignContentsHandler) Handle(
	_ context.Context,
	cmd contentcmd.PresignContentsCommand,
) (*contentcmd.PresignContentsResult, error) {
	m.called = true
	m.cmd = cmd
	return m.result, m.err
}

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
	presign *mockPresignContentsHandler,
	getByID *mockGetContentByIDHandler,
	list *mockListContentsByCampaignHandler,
	stats *mockContentStatsByClientHandler,
	storage *mockContentStorage,
) *handler.ContentHandler {
	if presign == nil {
		presign = &mockPresignContentsHandler{}
	}
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
	return handler.NewContentHandler(presign, getByID, list, stats, storage)
}

func sampleContentView() *domaincontent.ContentView {
	size := int64(1024)
	thumb := "clients/x/campaigns/y/contents/thumbnails/z.jpg"
	return &domaincontent.ContentView{
		ID:           uuid.MustParse("01960000-0000-7000-8000-0000000000c1"),
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

func TestContentHandler_Presign_Success(t *testing.T) {
	contentID := uuid.MustParse("01960000-0000-7000-8000-0000000000c1")
	presign := &mockPresignContentsHandler{
		result: &contentcmd.PresignContentsResult{
			CampaignID: testutil.TestCampaignID,
			Items: []contentcmd.PresignContentItem{{
				ContentID: contentID,
				URL:       "https://minio.example/upload",
				ObjectKey: "clients/x/campaigns/y/contents/z.png",
				Filename:  "shot.png",
			}},
		},
	}
	h := newContentHandler(presign, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/contents/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost, "/contents/presign", map[string]any{
		"campaignId": testutil.TestCampaignID.String(),
		"files": []map[string]any{
			{"filename": "shot.png", "contentType": "image/png"},
		},
	}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if !presign.called || presign.cmd.CampaignID != testutil.TestCampaignID || len(presign.cmd.Files) != 1 {
		t.Fatalf("unexpected cmd: %+v", presign.cmd)
	}
	body := testutil.DecodeJSONMap(t, resp)
	if body["campaignId"] != testutil.TestCampaignID.String() {
		t.Fatalf("campaignId: %#v", body["campaignId"])
	}
	items, _ := body["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items: %#v", body["items"])
	}
}

func TestContentHandler_Presign_EmptyCampaignID_UsesDefault(t *testing.T) {
	presign := &mockPresignContentsHandler{
		result: &contentcmd.PresignContentsResult{
			CampaignID: testutil.TestCampaignID,
			Items: []contentcmd.PresignContentItem{{
				ContentID: uuid.MustParse("01960000-0000-7000-8000-0000000000c1"),
				URL:       "https://minio.example/upload",
				ObjectKey: "key",
				Filename:  "a.png",
			}},
		},
	}
	h := newContentHandler(presign, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/contents/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost, "/contents/presign", map[string]any{
		"files": []map[string]any{{"filename": "a.png"}},
	}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if !presign.called || presign.cmd.CampaignID != uuid.Nil {
		t.Fatalf("expected nil campaign id, got %+v", presign.cmd.CampaignID)
	}
}

func TestContentHandler_Presign_Unauthorized(t *testing.T) {
	h := newContentHandler(nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post("/contents/presign", h.Presign)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost, "/contents/presign",
		map[string]any{"files": []map[string]any{{"filename": "a.png"}}},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestContentHandler_Presign_MissingClient(t *testing.T) {
	h := newContentHandler(nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/contents/presign",
		testutil.WithUserWithoutClient(testutil.TestUserID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost, "/contents/presign",
		map[string]any{"files": []map[string]any{{"filename": "a.png"}}},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestContentHandler_Presign_UnsupportedMediaType(t *testing.T) {
	presign := &mockPresignContentsHandler{err: domaincontent.ErrUnsupportedContentType}
	h := newContentHandler(presign, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/contents/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost, "/contents/presign",
		map[string]any{"files": []map[string]any{{"filename": "a.pdf"}}},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestContentHandler_Presign_TooManyFiles(t *testing.T) {
	presign := &mockPresignContentsHandler{err: domaincontent.ErrTooManyFiles}
	h := newContentHandler(presign, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/contents/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost, "/contents/presign",
		map[string]any{"files": []map[string]any{{"filename": "a.png"}}},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestContentHandler_Presign_NotFound(t *testing.T) {
	presign := &mockPresignContentsHandler{err: errors.New("campaign not found")}
	h := newContentHandler(presign, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/contents/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost, "/contents/presign",
		map[string]any{"files": []map[string]any{{"filename": "a.png"}}},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestContentHandler_Presign_InvalidCampaignID(t *testing.T) {
	h := newContentHandler(nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/contents/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost, "/contents/presign", map[string]any{
		"campaignId": "bad",
		"files":      []map[string]any{{"filename": "a.png"}},
	}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestContentHandler_Presign_InvalidBody(t *testing.T) {
	h := newContentHandler(nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/contents/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost, "/contents/presign",
		map[string]any{"files": []any{}},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestContentHandler_Presign_Internal(t *testing.T) {
	presign := &mockPresignContentsHandler{err: errors.New("boom")}
	h := newContentHandler(presign, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/contents/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost, "/contents/presign",
		map[string]any{"files": []map[string]any{{"filename": "a.png"}}},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
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
	h := newContentHandler(nil, getByID, nil, nil, storage)
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
	h := newContentHandler(nil, getByID, nil, nil, nil)
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
	h := newContentHandler(nil, getByID, nil, nil, nil)
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
	h := newContentHandler(nil, nil, nil, nil, nil)
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
	h := newContentHandler(nil, getByID, nil, nil, nil)
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

func TestContentHandler_Presign_EmptyFileListError(t *testing.T) {
	presign := &mockPresignContentsHandler{err: domaincontent.ErrEmptyFileList}
	h := newContentHandler(presign, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/contents/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost, "/contents/presign",
		map[string]any{"files": []map[string]any{{"filename": "a.png"}}},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
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
	h := newContentHandler(nil, nil, list, nil, nil)
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
	h := newContentHandler(nil, nil, list, nil, nil)
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
	h := newContentHandler(nil, nil, list, nil, nil)
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
	h := newContentHandler(nil, nil, nil, nil, nil)
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
	h := newContentHandler(nil, nil, nil, nil, nil)
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
	h := newContentHandler(nil, nil, nil, nil, nil)
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
	h := newContentHandler(nil, nil, list, nil, nil)
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
	h := newContentHandler(nil, nil, list, nil, nil)
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
	h := newContentHandler(nil, getByID, nil, nil, nil)
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
	h := newContentHandler(nil, nil, nil, nil, nil)
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
	h := newContentHandler(nil, nil, nil, nil, nil)
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
	h := newContentHandler(nil, nil, nil, nil, nil)
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
	h := newContentHandler(nil, getByID, nil, nil, nil)
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
	h := newContentHandler(nil, getByID, nil, nil, nil)
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
	h := newContentHandler(nil, getByID, nil, nil, nil)
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
			Failed:        1,
			Human:         2,
			AIGenerated:   3,
			Uncertain:     4,
		},
	}
	h := newContentHandler(nil, nil, nil, stats, nil)
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
		body["analyzing"] != float64(7) {
		t.Fatalf("body: %#v", body)
	}
}

func TestContentHandler_Stats_Unauthorized(t *testing.T) {
	h := newContentHandler(nil, nil, nil, nil, nil)
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
	h := newContentHandler(nil, nil, nil, nil, nil)
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
	h := newContentHandler(nil, nil, nil, stats, nil)
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
