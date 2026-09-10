package campaigntest

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	campaigncmd "go-api/internal/application/command/campaign"
	domaincampaign "go-api/internal/domain/campaign"
	"go-api/internal/interfaces/http/testutil"
)

func TestCampaignHandler_PresignBackground_Success(t *testing.T) {
	presign := &mockPresignBackgroundHandler{
		result: &campaigncmd.PresignBackgroundResult{URL: "https://minio.example/upload"},
	}
	h := newCampaignHandlerWithExtras(nil, nil, nil, nil, nil, nil, presign, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/campaigns/:id/background/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.PresignBackground,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/campaigns/"+testutil.TestCampaignID.String()+"/background/presign",
		map[string]any{"filename": "hero.png", "contentType": "image/png"},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if !presign.called {
		t.Fatal("expected presign handler to be called")
	}
	if presign.cmd.CampaignID != testutil.TestCampaignID || presign.cmd.ClientID != testutil.TestClientID {
		t.Fatalf("unexpected command ids: %+v", presign.cmd)
	}
	body := testutil.DecodeJSONMap(t, resp)
	if body["url"] != "https://minio.example/upload" {
		t.Fatalf("url: got %#v", body["url"])
	}
}

func TestCampaignHandler_PresignBackground_Unauthorized(t *testing.T) {
	h := newCampaignHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post("/campaigns/:id/background/presign", h.PresignBackground)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/campaigns/"+testutil.TestCampaignID.String()+"/background/presign",
		map[string]any{"filename": "hero.png"},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_PresignBackground_MissingClient(t *testing.T) {
	h := newCampaignHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/campaigns/:id/background/presign",
		testutil.WithUserWithoutClient(testutil.TestUserID),
		h.PresignBackground,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/campaigns/"+testutil.TestCampaignID.String()+"/background/presign",
		map[string]any{"filename": "hero.png"},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_PresignBackground_WrongClient(t *testing.T) {
	presign := &mockPresignBackgroundHandler{err: errors.New("campaign not found")}
	h := newCampaignHandlerWithExtras(nil, nil, nil, nil, nil, nil, presign, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/campaigns/:id/background/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.PresignBackground,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/campaigns/"+testutil.TestCampaignID.String()+"/background/presign",
		map[string]any{"filename": "hero.png"},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_PresignBackground_UnsupportedMediaType(t *testing.T) {
	presign := &mockPresignBackgroundHandler{err: domaincampaign.ErrUnsupportedBackgroundType}
	h := newCampaignHandlerWithExtras(nil, nil, nil, nil, nil, nil, presign, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/campaigns/:id/background/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.PresignBackground,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/campaigns/"+testutil.TestCampaignID.String()+"/background/presign",
		map[string]any{"filename": "notes.pdf"},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_PresignBackground_NotFound(t *testing.T) {
	presign := &mockPresignBackgroundHandler{err: errors.New("campaign not found")}
	h := newCampaignHandlerWithExtras(nil, nil, nil, nil, nil, nil, presign, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/campaigns/:id/background/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.PresignBackground,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/campaigns/"+testutil.TestCampaignID.String()+"/background/presign",
		map[string]any{"filename": "hero.png"},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_PresignBackground_InvalidID(t *testing.T) {
	h := newCampaignHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/campaigns/:id/background/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.PresignBackground,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/campaigns/bad/background/presign",
		map[string]any{"filename": "hero.png"},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_PresignBackground_InvalidBody(t *testing.T) {
	h := newCampaignHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/campaigns/:id/background/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.PresignBackground,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/campaigns/"+testutil.TestCampaignID.String()+"/background/presign",
		map[string]any{"filename": ""},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_PresignBackground_HandlerError_Internal(t *testing.T) {
	presign := &mockPresignBackgroundHandler{err: errors.New("boom")}
	h := newCampaignHandlerWithExtras(nil, nil, nil, nil, nil, nil, presign, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/campaigns/:id/background/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.PresignBackground,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/campaigns/"+testutil.TestCampaignID.String()+"/background/presign",
		map[string]any{"filename": "hero.png"},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_GetThumbnail_Success(t *testing.T) {
	thumbKey := domaincampaign.NewBackgroundThumbnailKey(testutil.TestClientID, testutil.TestCampaignID)
	view := sampleCampaignView()
	view.BackgroundStatus = domaincampaign.BackgroundStatusReady
	view.BackgroundThumbnailKey = thumbKey
	getByID := &mockGetCampaignByIDHandler{views: []*domaincampaign.CampaignView{view}, errs: []error{nil}}
	storage := &mockCampaignStorage{body: []byte{0xff, 0xd8, 0xff}}
	h := newCampaignHandlerWithExtras(nil, nil, nil, getByID, nil, nil, nil, nil, storage)
	app := testutil.NewTestApp()
	app.Get(
		"/campaigns/:id/thumbnail",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.GetThumbnail,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/campaigns/"+testutil.TestCampaignID.String()+"/thumbnail",
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/jpeg" {
		t.Fatalf("content-type: got %q", ct)
	}
	if !storage.called || storage.key != thumbKey {
		t.Fatalf("storage call: called=%v key=%q", storage.called, storage.key)
	}
}

func TestCampaignHandler_GetThumbnail_Missing(t *testing.T) {
	view := sampleCampaignView()
	view.BackgroundStatus = domaincampaign.BackgroundStatusNone
	getByID := &mockGetCampaignByIDHandler{views: []*domaincampaign.CampaignView{view}, errs: []error{nil}}
	h := newCampaignHandlerWithExtras(nil, nil, nil, getByID, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get(
		"/campaigns/:id/thumbnail",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.GetThumbnail,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/campaigns/"+testutil.TestCampaignID.String()+"/thumbnail",
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_GetThumbnail_WrongClient(t *testing.T) {
	view := sampleCampaignView()
	view.ClientID = otherClientID
	view.BackgroundStatus = domaincampaign.BackgroundStatusReady
	view.BackgroundThumbnailKey = "thumb.jpg"
	getByID := &mockGetCampaignByIDHandler{views: []*domaincampaign.CampaignView{view}, errs: []error{nil}}
	h := newCampaignHandlerWithExtras(nil, nil, nil, getByID, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get(
		"/campaigns/:id/thumbnail",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.GetThumbnail,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/campaigns/"+testutil.TestCampaignID.String()+"/thumbnail",
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_GetThumbnail_Unauthorized(t *testing.T) {
	h := newCampaignHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/campaigns/:id/thumbnail", h.GetThumbnail)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/campaigns/"+testutil.TestCampaignID.String()+"/thumbnail",
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_GetThumbnail_MissingClient(t *testing.T) {
	h := newCampaignHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get(
		"/campaigns/:id/thumbnail",
		testutil.WithUserWithoutClient(testutil.TestUserID),
		h.GetThumbnail,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/campaigns/"+testutil.TestCampaignID.String()+"/thumbnail",
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_GetThumbnail_NotFound(t *testing.T) {
	getByID := &mockGetCampaignByIDHandler{
		views: []*domaincampaign.CampaignView{nil},
		errs:  []error{errors.New("campaign not found")},
	}
	h := newCampaignHandlerWithExtras(nil, nil, nil, getByID, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get(
		"/campaigns/:id/thumbnail",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.GetThumbnail,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/campaigns/"+testutil.TestCampaignID.String()+"/thumbnail",
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_GetThumbnail_StorageError(t *testing.T) {
	view := sampleCampaignView()
	view.BackgroundStatus = domaincampaign.BackgroundStatusReady
	view.BackgroundThumbnailKey = "thumb.jpg"
	getByID := &mockGetCampaignByIDHandler{views: []*domaincampaign.CampaignView{view}, errs: []error{nil}}
	storage := &mockCampaignStorage{err: errors.New("missing")}
	h := newCampaignHandlerWithExtras(nil, nil, nil, getByID, nil, nil, nil, nil, storage)
	app := testutil.NewTestApp()
	app.Get(
		"/campaigns/:id/thumbnail",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.GetThumbnail,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/campaigns/"+testutil.TestCampaignID.String()+"/thumbnail",
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_GetThumbnail_ReadError(t *testing.T) {
	view := sampleCampaignView()
	view.BackgroundStatus = domaincampaign.BackgroundStatusReady
	view.BackgroundThumbnailKey = "thumb.jpg"
	getByID := &mockGetCampaignByIDHandler{views: []*domaincampaign.CampaignView{view}, errs: []error{nil}}
	h := newCampaignHandlerWithExtras(nil, nil, nil, getByID, nil, nil, nil, nil, &failingReadStorage{})
	app := testutil.NewTestApp()
	app.Get(
		"/campaigns/:id/thumbnail",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.GetThumbnail,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/campaigns/"+testutil.TestCampaignID.String()+"/thumbnail",
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_GetThumbnail_InvalidID(t *testing.T) {
	h := newCampaignHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get(
		"/campaigns/:id/thumbnail",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.GetThumbnail,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/campaigns/bad/thumbnail", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_GetThumbnail_GetInternal(t *testing.T) {
	getByID := &mockGetCampaignByIDHandler{
		views: []*domaincampaign.CampaignView{nil},
		errs:  []error{errors.New("db")},
	}
	h := newCampaignHandlerWithExtras(nil, nil, nil, getByID, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get(
		"/campaigns/:id/thumbnail",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.GetThumbnail,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/campaigns/"+testutil.TestCampaignID.String()+"/thumbnail",
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_GetThumbnail_WhilePendingReplace(t *testing.T) {
	thumbKey := domaincampaign.NewBackgroundThumbnailKey(testutil.TestClientID, testutil.TestCampaignID)
	view := sampleCampaignView()
	view.BackgroundStatus = domaincampaign.BackgroundStatusPending
	view.BackgroundThumbnailKey = thumbKey
	getByID := &mockGetCampaignByIDHandler{views: []*domaincampaign.CampaignView{view}, errs: []error{nil}}
	storage := &mockCampaignStorage{body: []byte{0xff, 0xd8, 0xff}}
	h := newCampaignHandlerWithExtras(nil, nil, nil, getByID, nil, nil, nil, nil, storage)
	app := testutil.NewTestApp()
	app.Get(
		"/campaigns/:id/thumbnail",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.GetThumbnail,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/campaigns/"+testutil.TestCampaignID.String()+"/thumbnail",
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_ClearBackground_Success(t *testing.T) {
	clear := &mockClearBackgroundHandler{}
	h := newCampaignHandlerWithExtras(nil, nil, nil, nil, nil, nil, nil, clear, nil)
	app := testutil.NewTestApp()
	app.Delete(
		"/campaigns/:id/background",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.ClearBackground,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodDelete,
		"/campaigns/"+testutil.TestCampaignID.String()+"/background",
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if !clear.called {
		t.Fatal("expected clear handler to be called")
	}
	if clear.cmd.CampaignID != testutil.TestCampaignID || clear.cmd.ClientID != testutil.TestClientID {
		t.Fatalf("unexpected command: %+v", clear.cmd)
	}
}

func TestCampaignHandler_ClearBackground_Unauthorized(t *testing.T) {
	h := newCampaignHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Delete("/campaigns/:id/background", h.ClearBackground)

	resp, err := app.Test(mustJSONRequest(t, http.MethodDelete,
		"/campaigns/"+testutil.TestCampaignID.String()+"/background",
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_ClearBackground_MissingClient(t *testing.T) {
	h := newCampaignHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Delete(
		"/campaigns/:id/background",
		testutil.WithUserWithoutClient(testutil.TestUserID),
		h.ClearBackground,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodDelete,
		"/campaigns/"+testutil.TestCampaignID.String()+"/background",
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_ClearBackground_NotFound(t *testing.T) {
	clear := &mockClearBackgroundHandler{err: errors.New("campaign not found")}
	h := newCampaignHandlerWithExtras(nil, nil, nil, nil, nil, nil, nil, clear, nil)
	app := testutil.NewTestApp()
	app.Delete(
		"/campaigns/:id/background",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.ClearBackground,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodDelete,
		"/campaigns/"+testutil.TestCampaignID.String()+"/background",
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_ClearBackground_InvalidID(t *testing.T) {
	h := newCampaignHandler(nil, nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Delete(
		"/campaigns/:id/background",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.ClearBackground,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodDelete, "/campaigns/bad/background", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestCampaignHandler_ClearBackground_HandlerError_Internal(t *testing.T) {
	clear := &mockClearBackgroundHandler{err: errors.New("boom")}
	h := newCampaignHandlerWithExtras(nil, nil, nil, nil, nil, nil, nil, clear, nil)
	app := testutil.NewTestApp()
	app.Delete(
		"/campaigns/:id/background",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.ClearBackground,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodDelete,
		"/campaigns/"+testutil.TestCampaignID.String()+"/background",
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

type failingReadStorage struct{}

func (f *failingReadStorage) GetThumbnail(_ context.Context, _ string) (io.ReadCloser, error) {
	return io.NopCloser(&errReader{}), nil
}

type errReader struct{}

func (e *errReader) Read(_ []byte) (int, error) {
	return 0, errors.New("read failed")
}
