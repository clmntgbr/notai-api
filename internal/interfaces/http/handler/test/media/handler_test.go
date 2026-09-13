package mediatest

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	mediacmd "go-api/internal/application/command/media"
	querymedia "go-api/internal/application/query/media"
	domaincampaign "go-api/internal/domain/campaign"
	domainmedia "go-api/internal/domain/media"
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
	query  querymedia.ListByClientQuery
	result *querymedia.ListByClientResult
	err    error
}

func (m *mockListMediaHandler) Handle(
	_ context.Context,
	q querymedia.ListByClientQuery,
) (*querymedia.ListByClientResult, error) {
	m.called = true
	m.query = q
	if m.result != nil {
		return m.result, m.err
	}
	return &querymedia.ListByClientResult{}, m.err
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

type mockMediaStatsHandler struct {
	called bool
	query  querymedia.GetStatsByClientQuery
	stats  *domainmedia.MediaStats
	err    error
}

func (m *mockMediaStatsHandler) Handle(
	_ context.Context,
	q querymedia.GetStatsByClientQuery,
) (*domainmedia.MediaStats, error) {
	m.called = true
	m.query = q
	return m.stats, m.err
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
	stats *mockMediaStatsHandler,
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
	if stats == nil {
		stats = &mockMediaStatsHandler{}
	}
	if storage == nil {
		storage = &mockMediaStorage{body: []byte("jpeg")}
	}
	return handler.NewMediaHandler(presign, list, getByID, stats, storage)
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
	h := newMediaHandler(presign, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/medias/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/medias/presign",
		map[string]any{
			"campaignId": testutil.TestCampaignID.String(),
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

func TestMediaHandler_Presign_DefaultCampaign(t *testing.T) {
	presign := &mockPresignMediaHandler{
		result: &mediacmd.PresignMediaResult{
			CampaignID: testutil.TestCampaignID,
			Items: []mediacmd.PresignMediaItem{{
				MediaID:   uuid.MustParse("01960000-0000-7000-8000-0000000000a1"),
				URL:       "https://minio.example/upload",
				ObjectKey: "key",
				Filename:  "a.png",
				MediaType: domainmedia.MediaTypeImage,
			}},
		},
	}
	h := newMediaHandler(presign, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post("/medias/presign", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Presign)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost, "/medias/presign",
		map[string]any{"files": []map[string]any{{"filename": "a.png", "contentType": "image/png"}}},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if !presign.called || presign.cmd.CampaignID != uuid.Nil {
		t.Fatalf("presign cmd: %+v", presign.cmd)
	}
}

func TestMediaHandler_Presign_Unauthorized(t *testing.T) {
	h := newMediaHandler(nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post("/medias/presign", h.Presign)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/medias/presign",
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
	h := newMediaHandler(nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/medias/presign",
		testutil.WithUserWithoutClient(testutil.TestUserID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/medias/presign",
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
	h := newMediaHandler(presign, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/medias/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/medias/presign",
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
	h := newMediaHandler(presign, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/medias/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/medias/presign",
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
	h := newMediaHandler(nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/medias/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/medias/presign",
		map[string]any{
			"campaignId": "not-a-uuid",
			"files":      []map[string]any{{"filename": "a.png", "contentType": "image/png"}},
		},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_Presign_InvalidBody(t *testing.T) {
	h := newMediaHandler(nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/medias/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/medias/presign",
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
	h := newMediaHandler(presign, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/medias/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/medias/presign",
		map[string]any{"files": []map[string]any{{"filename": "a.png", "contentType": "image/png"}}},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
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
	h := newMediaHandler(nil, nil, getByID, nil, nil)
	app := testutil.NewTestApp()
	app.Get(
		"/medias/:id",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.GetByID,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/medias/"+sampleMediaView().ID.String(),
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
	h := newMediaHandler(nil, nil, getByID, nil, nil)
	app := testutil.NewTestApp()
	app.Get(
		"/medias/:id",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.GetByID,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/medias/"+uuid.New().String(),
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
	h := newMediaHandler(nil, nil, getByID, nil, storage)
	app := testutil.NewTestApp()
	app.Get(
		"/medias/:id/thumbnail",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.GetThumbnail,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/medias/"+sampleMediaView().ID.String()+"/thumbnail",
		nil,
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_Stats_Success(t *testing.T) {
	verificationsChange := 18.4
	authChange := 4.2
	toReviewChange := -12.0
	stats := &mockMediaStatsHandler{
		stats: &domainmedia.MediaStats{
			PendingUpload: 1,
			Uploaded:      2,
			Processing:    3,
			Analyzed:      4,
			Failed:        5,
			Human:         6,
			AIGenerated:   7,
			Uncertain:     8,
			MonthlyControls: []domainmedia.MediaMonthlyStats{
				{Month: "2026-04", Analyzed: 1, Human: 1},
				{Month: "2026-09", Analyzed: 3, AIGenerated: 2, Uncertain: 1},
			},
			KPIs: domainmedia.MediaDashboardKPIs{
				Month:                      "2026-09",
				Verifications:              478,
				VerificationsChangePercent: &verificationsChange,
				AuthenticityRatePercent:    88.1,
				AuthenticityChangePoints:   &authChange,
				ValidatedCount:             421,
				ToReviewCount:              38,
				ToReviewChangePercent:      &toReviewChange,
				AIGeneratedCount:           19,
				AIGeneratedSharePercent:    4.0,
			},
		},
	}
	h := newMediaHandler(nil, nil, nil, stats, nil)
	app := testutil.NewTestApp()
	app.Get("/medias/stats", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Stats)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/medias/stats", nil))
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
	if body["pendingUpload"] != float64(1) || body["uploaded"] != float64(2) ||
		body["processing"] != float64(3) || body["analyzed"] != float64(4) ||
		body["failed"] != float64(5) || body["human"] != float64(6) ||
		body["aiGenerated"] != float64(7) || body["uncertain"] != float64(8) {
		t.Fatalf("body: %#v", body)
	}
	monthly, ok := body["monthlyControls"].([]any)
	if !ok || len(monthly) != 2 {
		t.Fatalf("monthlyControls: %#v", body["monthlyControls"])
	}
	kpis, ok := body["kpis"].(map[string]any)
	if !ok {
		t.Fatalf("kpis: %#v", body["kpis"])
	}
	if kpis["verifications"] != float64(478) ||
		kpis["verificationsChangePercent"] != 18.4 ||
		kpis["authenticityRatePercent"] != 88.1 ||
		kpis["authenticityChangePoints"] != 4.2 ||
		kpis["validatedCount"] != float64(421) ||
		kpis["toReviewCount"] != float64(38) ||
		kpis["toReviewChangePercent"] != -12.0 ||
		kpis["aiGeneratedCount"] != float64(19) ||
		kpis["aiGeneratedSharePercent"] != 4.0 ||
		kpis["planIncluded"] != nil {
		t.Fatalf("kpis: %#v", kpis)
	}
}

func TestMediaHandler_Stats_Unauthorized(t *testing.T) {
	h := newMediaHandler(nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/medias/stats", h.Stats)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/medias/stats", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_Stats_MissingClient(t *testing.T) {
	h := newMediaHandler(nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/medias/stats", testutil.WithUserWithoutClient(testutil.TestUserID), h.Stats)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/medias/stats", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_Stats_Internal(t *testing.T) {
	stats := &mockMediaStatsHandler{err: errors.New("boom")}
	h := newMediaHandler(nil, nil, nil, stats, nil)
	app := testutil.NewTestApp()
	app.Get("/medias/stats", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.Stats)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/medias/stats", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_GetByID_Unauthorized(t *testing.T) {
	h := newMediaHandler(nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/medias/:id", h.GetByID)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/medias/"+sampleMediaView().ID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_GetByID_MissingClient(t *testing.T) {
	h := newMediaHandler(nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/medias/:id", testutil.WithUserWithoutClient(testutil.TestUserID), h.GetByID)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/medias/"+sampleMediaView().ID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_GetByID_InvalidID(t *testing.T) {
	h := newMediaHandler(nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/medias/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetByID)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/medias/not-a-uuid", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_GetByID_Internal(t *testing.T) {
	getByID := &mockGetMediaByIDHandler{err: errors.New("boom")}
	h := newMediaHandler(nil, nil, getByID, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/medias/:id", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetByID)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/medias/"+sampleMediaView().ID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_GetThumbnail_Unauthorized(t *testing.T) {
	h := newMediaHandler(nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/medias/:id/thumbnail", h.GetThumbnail)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/medias/"+sampleMediaView().ID.String()+"/thumbnail", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_GetThumbnail_MissingClient(t *testing.T) {
	h := newMediaHandler(nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/medias/:id/thumbnail", testutil.WithUserWithoutClient(testutil.TestUserID), h.GetThumbnail)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/medias/"+sampleMediaView().ID.String()+"/thumbnail", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_GetThumbnail_InvalidID(t *testing.T) {
	h := newMediaHandler(nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/medias/:id/thumbnail", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetThumbnail)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/medias/not-a-uuid/thumbnail", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_GetThumbnail_NotFound(t *testing.T) {
	getByID := &mockGetMediaByIDHandler{err: errors.New("media not found")}
	h := newMediaHandler(nil, nil, getByID, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/medias/:id/thumbnail", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetThumbnail)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/medias/"+sampleMediaView().ID.String()+"/thumbnail", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_GetThumbnail_StorageError(t *testing.T) {
	getByID := &mockGetMediaByIDHandler{result: &querymedia.GetByIDResult{Media: sampleMediaView()}}
	storage := &mockMediaStorage{err: errors.New("missing")}
	h := newMediaHandler(nil, nil, getByID, nil, storage)
	app := testutil.NewTestApp()
	app.Get("/medias/:id/thumbnail", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetThumbnail)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/medias/"+sampleMediaView().ID.String()+"/thumbnail", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_GetContentThumbnail_Success(t *testing.T) {
	contentID := uuid.MustParse("01960000-0000-7000-8000-0000000000c1")
	thumbKey := "clients/x/campaigns/y/media/thumbnails/contents/c1.jpg"
	getByID := &mockGetMediaByIDHandler{
		result: &querymedia.GetByIDResult{
			Media: sampleMediaView(),
			Contents: []domainmedia.ContentChildView{{
				ID:           contentID,
				MediaID:      sampleMediaView().ID,
				ThumbnailKey: &thumbKey,
				Status:       "analyzed",
				UpdatedAt:    time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
			}},
		},
	}
	storage := &mockMediaStorage{body: []byte("frame-jpeg")}
	h := newMediaHandler(nil, nil, getByID, nil, storage)
	app := testutil.NewTestApp()
	app.Get(
		"/medias/:id/contents/:contentId/thumbnail",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.GetContentThumbnail,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/medias/"+sampleMediaView().ID.String()+"/contents/"+contentID.String()+"/thumbnail", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_GetContentThumbnail_MissingThumbnail(t *testing.T) {
	contentID := uuid.MustParse("01960000-0000-7000-8000-0000000000c1")
	getByID := &mockGetMediaByIDHandler{
		result: &querymedia.GetByIDResult{
			Media: sampleMediaView(),
			Contents: []domainmedia.ContentChildView{{
				ID:     contentID,
				Status: "uploaded",
			}},
		},
	}
	h := newMediaHandler(nil, nil, getByID, nil, nil)
	app := testutil.NewTestApp()
	app.Get(
		"/medias/:id/contents/:contentId/thumbnail",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.GetContentThumbnail,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/medias/"+sampleMediaView().ID.String()+"/contents/"+contentID.String()+"/thumbnail", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_GetContentThumbnail_Unauthorized(t *testing.T) {
	h := newMediaHandler(nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/medias/:id/contents/:contentId/thumbnail", h.GetContentThumbnail)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/medias/"+sampleMediaView().ID.String()+"/contents/"+uuid.New().String()+"/thumbnail", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_GetContentThumbnail_MissingClient(t *testing.T) {
	h := newMediaHandler(nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get(
		"/medias/:id/contents/:contentId/thumbnail",
		testutil.WithUserWithoutClient(testutil.TestUserID),
		h.GetContentThumbnail,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/medias/"+sampleMediaView().ID.String()+"/contents/"+uuid.New().String()+"/thumbnail", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_GetThumbnail_Internal(t *testing.T) {
	getByID := &mockGetMediaByIDHandler{err: errors.New("boom")}
	h := newMediaHandler(nil, nil, getByID, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/medias/:id/thumbnail", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetThumbnail)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/medias/"+sampleMediaView().ID.String()+"/thumbnail", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_GetContentThumbnail_InvalidMediaID(t *testing.T) {
	h := newMediaHandler(nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get(
		"/medias/:id/contents/:contentId/thumbnail",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.GetContentThumbnail,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/medias/not-a-uuid/contents/"+uuid.New().String()+"/thumbnail", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_GetContentThumbnail_InvalidContentID(t *testing.T) {
	h := newMediaHandler(nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get(
		"/medias/:id/contents/:contentId/thumbnail",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.GetContentThumbnail,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/medias/"+sampleMediaView().ID.String()+"/contents/not-a-uuid/thumbnail", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_GetContentThumbnail_MediaNotFound(t *testing.T) {
	getByID := &mockGetMediaByIDHandler{err: errors.New("media not found")}
	h := newMediaHandler(nil, nil, getByID, nil, nil)
	app := testutil.NewTestApp()
	app.Get(
		"/medias/:id/contents/:contentId/thumbnail",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.GetContentThumbnail,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/medias/"+sampleMediaView().ID.String()+"/contents/"+uuid.New().String()+"/thumbnail", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_GetContentThumbnail_Internal(t *testing.T) {
	getByID := &mockGetMediaByIDHandler{err: errors.New("boom")}
	h := newMediaHandler(nil, nil, getByID, nil, nil)
	app := testutil.NewTestApp()
	app.Get(
		"/medias/:id/contents/:contentId/thumbnail",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.GetContentThumbnail,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/medias/"+sampleMediaView().ID.String()+"/contents/"+uuid.New().String()+"/thumbnail", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_GetContentThumbnail_ContentNotFound(t *testing.T) {
	getByID := &mockGetMediaByIDHandler{
		result: &querymedia.GetByIDResult{Media: sampleMediaView(), Contents: nil},
	}
	h := newMediaHandler(nil, nil, getByID, nil, nil)
	app := testutil.NewTestApp()
	app.Get(
		"/medias/:id/contents/:contentId/thumbnail",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.GetContentThumbnail,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/medias/"+sampleMediaView().ID.String()+"/contents/"+uuid.New().String()+"/thumbnail", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_GetContentThumbnail_StorageError(t *testing.T) {
	contentID := uuid.MustParse("01960000-0000-7000-8000-0000000000c1")
	thumbKey := "thumb.jpg"
	getByID := &mockGetMediaByIDHandler{
		result: &querymedia.GetByIDResult{
			Media: sampleMediaView(),
			Contents: []domainmedia.ContentChildView{{
				ID: contentID, ThumbnailKey: &thumbKey, Status: "analyzed",
			}},
		},
	}
	storage := &mockMediaStorage{err: errors.New("missing")}
	h := newMediaHandler(nil, nil, getByID, nil, storage)
	app := testutil.NewTestApp()
	app.Get(
		"/medias/:id/contents/:contentId/thumbnail",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.GetContentThumbnail,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/medias/"+sampleMediaView().ID.String()+"/contents/"+contentID.String()+"/thumbnail", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_Presign_BadRequest(t *testing.T) {
	presign := &mockPresignMediaHandler{err: domainmedia.ErrInvalidFilename}
	h := newMediaHandler(presign, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Post(
		"/medias/presign",
		testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID),
		h.Presign,
	)

	resp, err := app.Test(mustJSONRequest(t, http.MethodPost,
		"/medias/presign",
		map[string]any{"files": []map[string]any{{"filename": "a.png", "contentType": "image/png"}}},
	))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_List_Success(t *testing.T) {
	view := sampleMediaView()
	campaign := &domaincampaign.CampaignView{
		ID:               view.CampaignID,
		ClientID:         view.ClientID,
		Name:             "Spring",
		IsDefault:        false,
		BackgroundStatus: domaincampaign.BackgroundStatusNone,
	}
	list := &mockListMediaHandler{
		result: &querymedia.ListByClientResult{
			Views: []domainmedia.MediaView{view},
			Total: 1,
			Campaigns: map[uuid.UUID]*domaincampaign.CampaignView{
				view.CampaignID: campaign,
			},
		},
	}
	h := newMediaHandler(nil, list, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/medias", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.List)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/medias?page=1&limit=10", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if !list.called || list.query.ClientID != testutil.TestClientID || list.query.CampaignID != uuid.Nil {
		t.Fatalf("list query: %+v", list.query)
	}
	body := testutil.DecodeJSONMap(t, resp)
	members, ok := body["members"].([]any)
	if !ok || len(members) != 1 {
		t.Fatalf("members: %#v", body["members"])
	}
	item, ok := members[0].(map[string]any)
	if !ok {
		t.Fatalf("item: %#v", members[0])
	}
	if item["id"] != view.ID.String() {
		t.Fatalf("id: %#v", item["id"])
	}
	thumb, _ := item["thumbnailUrl"].(string)
	if !strings.HasPrefix(thumb, "/api/medias/"+view.ID.String()+"/thumbnail") {
		t.Fatalf("thumbnailUrl: %#v", item["thumbnailUrl"])
	}
	camp, ok := item["campaign"].(map[string]any)
	if !ok || camp["name"] != "Spring" || camp["id"] != view.CampaignID.String() {
		t.Fatalf("campaign: %#v", item["campaign"])
	}
}

func TestMediaHandler_List_DefaultCampaignOmitsCampaign(t *testing.T) {
	view := sampleMediaView()
	campaign := &domaincampaign.CampaignView{
		ID:        view.CampaignID,
		ClientID:  view.ClientID,
		Name:      "Default",
		IsDefault: true,
	}
	list := &mockListMediaHandler{
		result: &querymedia.ListByClientResult{
			Views: []domainmedia.MediaView{view},
			Total: 1,
			Campaigns: map[uuid.UUID]*domaincampaign.CampaignView{
				view.CampaignID: campaign,
			},
		},
	}
	h := newMediaHandler(nil, list, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/medias", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.List)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/medias", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	body := testutil.DecodeJSONMap(t, resp)
	members, ok := body["members"].([]any)
	if !ok || len(members) != 1 {
		t.Fatalf("members: %#v", body["members"])
	}
	item, ok := members[0].(map[string]any)
	if !ok {
		t.Fatalf("item: %#v", members[0])
	}
	if _, exists := item["campaign"]; exists {
		t.Fatalf("expected campaign omitted, got %#v", item["campaign"])
	}
}

func TestMediaHandler_List_WithCampaignFilter(t *testing.T) {
	list := &mockListMediaHandler{
		result: &querymedia.ListByClientResult{Views: []domainmedia.MediaView{}, Total: 0},
	}
	h := newMediaHandler(nil, list, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/medias", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.List)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/medias?campaignId="+testutil.TestCampaignID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	if !list.called || list.query.CampaignID != testutil.TestCampaignID {
		t.Fatalf("list query: %+v", list.query)
	}
}

func TestMediaHandler_List_Unauthorized(t *testing.T) {
	h := newMediaHandler(nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/medias", h.List)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/medias", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_List_MissingClient(t *testing.T) {
	h := newMediaHandler(nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/medias", testutil.WithUserWithoutClient(testutil.TestUserID), h.List)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/medias", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_List_InvalidCampaignID(t *testing.T) {
	h := newMediaHandler(nil, nil, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/medias", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.List)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/medias?campaignId=not-a-uuid", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_List_CampaignNotFound(t *testing.T) {
	list := &mockListMediaHandler{err: errors.New("campaign not found")}
	h := newMediaHandler(nil, list, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/medias", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.List)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet,
		"/medias?campaignId="+testutil.TestCampaignID.String(), nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaHandler_List_Internal(t *testing.T) {
	list := &mockListMediaHandler{err: errors.New("boom")}
	h := newMediaHandler(nil, list, nil, nil, nil)
	app := testutil.NewTestApp()
	app.Get("/medias", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.List)

	resp, err := app.Test(mustJSONRequest(t, http.MethodGet, "/medias", nil))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}
