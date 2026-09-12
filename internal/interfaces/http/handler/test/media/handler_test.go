package mediatest

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	mediacmd "go-api/internal/application/command/media"
	querymedia "go-api/internal/application/query/media"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/paginate"
	"go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/testutil"

	"github.com/google/uuid"
)

type mockPresignMediaHandler struct {
	called bool
	cmd    mediacmd.PresignMediaCommand
	result *mediacmd.PresignMediaResult
	err    error
}

func (m *mockPresignMediaHandler) Handle(
	_ context.Context,
	cmd mediacmd.PresignMediaCommand,
) (*mediacmd.PresignMediaResult, error) {
	m.called = true
	m.cmd = cmd
	return m.result, m.err
}

type mockListMediaHandler struct {
	called bool
	query  querymedia.ListByCampaignQuery
	result *querymedia.ListByCampaignResult
	err    error
}

func (m *mockListMediaHandler) Handle(
	_ context.Context,
	q querymedia.ListByCampaignQuery,
) (*querymedia.ListByCampaignResult, error) {
	m.called = true
	m.query = q
	if m.result != nil {
		return m.result, m.err
	}
	return &querymedia.ListByCampaignResult{}, m.err
}

type mockGetMediaByIDHandler struct {
	result *querymedia.GetByIDResult
	err    error
}

func (m *mockGetMediaByIDHandler) Handle(
	_ context.Context,
	_ querymedia.GetByIDQuery,
) (*querymedia.GetByIDResult, error) {
	return m.result, m.err
}

type mockMediaStorage struct {
	body []byte
	err  error
}

func (m *mockMediaStorage) GetThumbnail(_ context.Context, _ string) (io.ReadCloser, error) {
	if m.err != nil {
		return nil, m.err
	}
	return io.NopCloser(bytes.NewReader(m.body)), nil
}

func newMediaHandler(
	presign *mockPresignMediaHandler,
	list *mockListMediaHandler,
	getByID *mockGetMediaByIDHandler,
	storage *mockMediaStorage,
) *handler.MediaHandler {
	if presign == nil {
		presign = &mockPresignMediaHandler{}
	}
	if list == nil {
		list = &mockListMediaHandler{}
	}
	if getByID == nil {
		getByID = &mockGetMediaByIDHandler{}
	}
	if storage == nil {
		storage = &mockMediaStorage{body: []byte("jpeg")}
	}
	return handler.NewMediaHandler(presign, list, getByID, storage)
}

func sampleMediaView() domainmedia.MediaView {
	return domainmedia.MediaView{
		ID:          uuid.MustParse("01960000-0000-7000-8000-0000000000a1"),
		CampaignID:  testutil.TestCampaignID,
		ClientID:    testutil.TestClientID,
		Filename:    "shot.png",
		ContentType: "image/png",
		MediaType:   domainmedia.MediaTypeImage,
		ObjectKey:   "clients/x/campaigns/y/media/z.png",
		Status:      domainmedia.StatusAnalyzed,
		CreatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
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

func TestMediaHandler_Presign_Success(t *testing.T) {
	mediaID := uuid.MustParse("01960000-0000-7000-8000-0000000000a1")
	presign := &mockPresignMediaHandler{
		result: &mediacmd.PresignMediaResult{
			CampaignID: testutil.TestCampaignID,
			Items: []mediacmd.PresignMediaItem{{
				MediaID:   mediaID,
				URL:       "https://minio.example/upload",
				ObjectKey: "key",
				Filename:  "a.png",
				MediaType: domainmedia.MediaTypeImage,
			}},
		},
	}
	h := newMediaHandler(presign, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/campaigns/:id/media/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/campaigns/"+testutil.TestCampaignID.String()+"/media/presign",
		map[string]any{
			"files": []map[string]any{
				{"filename": "a.png", "contentType": "image/png"},
			},
		},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if !presign.called ||
		presign.cmd.CampaignID != testutil.TestCampaignID ||
		presign.cmd.ClientID != testutil.TestClientID {
		t.Fatalf("presign cmd: %+v", presign.cmd)
	}
}

func TestMediaHandler_Presign_Unauthorized(t *testing.T) {
	h := newMediaHandler(nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post("/campaigns/:id/media/presign", h.Presign)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/campaigns/"+testutil.TestCampaignID.String()+"/media/presign",
		map[string]any{"files": []map[string]any{{"filename": "a.png", "contentType": "image/png"}}},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_Presign_MissingClient(t *testing.T) {
	h := newMediaHandler(nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/campaigns/:id/media/presign",
		testutil.WithUserWithoutClient(testutil.TestUserID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/campaigns/"+testutil.TestCampaignID.String()+"/media/presign",
		map[string]any{"files": []map[string]any{{"filename": "a.png", "contentType": "image/png"}}},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_Presign_UnsupportedMediaType(t *testing.T) {
	presign := &mockPresignMediaHandler{err: domainmedia.ErrUnsupportedContentType}
	h := newMediaHandler(presign, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/campaigns/:id/media/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/campaigns/"+testutil.TestCampaignID.String()+"/media/presign",
		map[string]any{"files": []map[string]any{{"filename": "a.gif", "contentType": "image/gif"}}},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_Presign_NotFound(t *testing.T) {
	presign := &mockPresignMediaHandler{err: errors.New("campaign not found")}
	h := newMediaHandler(presign, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/campaigns/:id/media/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/campaigns/"+testutil.TestCampaignID.String()+"/media/presign",
		map[string]any{"files": []map[string]any{{"filename": "a.png", "contentType": "image/png"}}},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_Presign_InvalidCampaignID(t *testing.T) {
	h := newMediaHandler(nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/campaigns/:id/media/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/campaigns/not-a-uuid/media/presign",
		map[string]any{"files": []map[string]any{{"filename": "a.png", "contentType": "image/png"}}},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_Presign_InvalidBody(t *testing.T) {
	h := newMediaHandler(nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/campaigns/:id/media/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/campaigns/"+testutil.TestCampaignID.String()+"/media/presign",
		map[string]any{},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_Presign_Internal(t *testing.T) {
	presign := &mockPresignMediaHandler{err: errors.New("boom")}
	h := newMediaHandler(presign, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/campaigns/:id/media/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/campaigns/"+testutil.TestCampaignID.String()+"/media/presign",
		map[string]any{"files": []map[string]any{{"filename": "a.png", "contentType": "image/png"}}},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_ListByCampaign_Success(t *testing.T) {
	list := &mockListMediaHandler{
		result: &querymedia.ListByCampaignResult{
			Views: []domainmedia.MediaView{sampleMediaView()},
			Total: 1,
		},
	}
	h := newMediaHandler(nil, list, nil, nil)
	app := testutil.NewTestApp()
	app.Get(
		"/campaigns/:id/media",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.ListByCampaign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/campaigns/"+testutil.TestCampaignID.String()+"/media?page=1&limit=10",
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if !list.called || list.query.CampaignID != testutil.TestCampaignID {
		t.Fatalf("list query: %+v", list.query)
	}
	_ = paginate.OrderByDesc
}

func TestMediaHandler_ListByCampaign_Unauthorized(t *testing.T) {
	h := newMediaHandler(nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/campaigns/:id/media", h.ListByCampaign)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/campaigns/"+testutil.TestCampaignID.String()+"/media",
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_GetByID_Success(t *testing.T) {
	frame := 0
	getByID := &mockGetMediaByIDHandler{
		result: &querymedia.GetByIDResult{
			Media: sampleMediaView(),
			Contents: []domainmedia.ContentChildView{{
				ID:         uuid.MustParse("01960000-0000-7000-8000-0000000000c1"),
				MediaID:    uuid.MustParse("01960000-0000-7000-8000-0000000000a1"),
				FrameIndex: &frame,
				Status:     "analyzed",
			}},
		},
	}
	h := newMediaHandler(nil, nil, getByID, nil)
	app := testutil.NewTestApp()
	app.Get(
		"/media/:id",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.GetByID,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/media/"+sampleMediaView().ID.String(),
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_GetByID_NotFound(t *testing.T) {
	getByID := &mockGetMediaByIDHandler{err: errors.New("media not found")}
	h := newMediaHandler(nil, nil, getByID, nil)
	app := testutil.NewTestApp()
	app.Get(
		"/media/:id",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.GetByID,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/media/"+uuid.New().String(),
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_GetThumbnail_Success(t *testing.T) {
	getByID := &mockGetMediaByIDHandler{
		result: &querymedia.GetByIDResult{Media: sampleMediaView()},
	}
	storage := &mockMediaStorage{body: []byte("jpeg-bytes")}
	h := newMediaHandler(nil, nil, getByID, storage)
	app := testutil.NewTestApp()
	app.Get(
		"/media/:id/thumbnail",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.GetThumbnail,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/media/"+sampleMediaView().ID.String()+"/thumbnail",
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}
