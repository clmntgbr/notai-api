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
	domaincontent "go-api/internal/domain/content"
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
	storage *mockContentStorage,
) *handler.ContentHandler {
	if presign == nil {
		presign = &mockPresignContentsHandler{}
	}
	if getByID == nil {
		getByID = &mockGetContentByIDHandler{}
	}
	if storage == nil {
		storage = &mockContentStorage{}
	}
	return handler.NewContentHandler(presign, getByID, storage)
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
			Items: []contentcmd.PresignContentItem{{
				ContentID: contentID,
				URL:       "https://minio.example/upload",
				ObjectKey: "clients/x/campaigns/y/contents/z.png",
				Filename:  "shot.png",
			}},
		},
	}
	h := newContentHandler(presign, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/campaigns/:id/contents/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/campaigns/"+testutil.TestCampaignID.String()+"/contents/presign",
		map[string]any{
			"files": []map[string]any{
				{"filename": "shot.png", "contentType": "image/png"},
			},
		},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if !presign.called || len(presign.cmd.Files) != 1 {
		t.Fatalf("unexpected cmd: %+v", presign.cmd)
	}
	body := testutil.DecodeJSONMap(t, resp)
	items, _ := body["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items: %#v", body["items"])
	}
}

func TestContentHandler_Presign_Unauthorized(t *testing.T) {
	h := newContentHandler(nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post("/campaigns/:id/contents/presign", h.Presign)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/campaigns/"+testutil.TestCampaignID.String()+"/contents/presign",
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
	h := newContentHandler(nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/campaigns/:id/contents/presign",
		testutil.WithUserWithoutClient(testutil.TestUserID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/campaigns/"+testutil.TestCampaignID.String()+"/contents/presign",
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
	h := newContentHandler(presign, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/campaigns/:id/contents/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/campaigns/"+testutil.TestCampaignID.String()+"/contents/presign",
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
	h := newContentHandler(presign, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/campaigns/:id/contents/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/campaigns/"+testutil.TestCampaignID.String()+"/contents/presign",
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
	h := newContentHandler(presign, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/campaigns/:id/contents/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/campaigns/"+testutil.TestCampaignID.String()+"/contents/presign",
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
	h := newContentHandler(nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/campaigns/:id/contents/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/campaigns/bad/contents/presign",
		map[string]any{"files": []map[string]any{{"filename": "a.png"}}},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestContentHandler_Presign_InvalidBody(t *testing.T) {
	h := newContentHandler(nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/campaigns/:id/contents/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/campaigns/"+testutil.TestCampaignID.String()+"/contents/presign",
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
	h := newContentHandler(presign, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/campaigns/:id/contents/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/campaigns/"+testutil.TestCampaignID.String()+"/contents/presign",
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
	h := newContentHandler(nil, getByID, storage)
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
	h := newContentHandler(nil, getByID, nil)
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
	h := newContentHandler(nil, getByID, nil)
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
	h := newContentHandler(nil, nil, nil)
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
	h := newContentHandler(nil, getByID, nil)
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
	h := newContentHandler(presign, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/campaigns/:id/contents/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/campaigns/"+testutil.TestCampaignID.String()+"/contents/presign",
		map[string]any{"files": []map[string]any{{"filename": "a.png"}}},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}
