package mediauploadwebhooktest

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/middleware"
	"go-api/internal/interfaces/http/testutil"
)

const testMediaBucket = "media"

type processCall struct {
	ObjectKey   string
	ContentType string
	Size        int64
}

type mockProcessHandler struct {
	mu      sync.Mutex
	calls   []processCall
	err     error
	waiters []chan struct{}
}

func (m *mockProcessHandler) Handle(
	_ context.Context,
	objectKey, contentType string,
	size int64,
) error {
	m.mu.Lock()
	m.calls = append(m.calls, processCall{
		ObjectKey:   objectKey,
		ContentType: contentType,
		Size:        size,
	})
	waiters := m.waiters
	m.waiters = nil
	err := m.err
	m.mu.Unlock()
	for _, w := range waiters {
		close(w)
	}
	return err
}

func (m *mockProcessHandler) waitCalled(t *testing.T) {
	t.Helper()
	ch := make(chan struct{})
	m.mu.Lock()
	if len(m.calls) > 0 {
		m.mu.Unlock()
		return
	}
	m.waiters = append(m.waiters, ch)
	m.mu.Unlock()

	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for process handler")
	}
}

func objectCreatedPayload(bucket, key string) map[string]any {
	return map[string]any{
		"Records": []map[string]any{
			{
				"eventName": "s3:ObjectCreated:Put",
				"s3": map[string]any{
					"bucket": map[string]any{"name": bucket},
					"object": map[string]any{
						"key":         key,
						"size":        int64(1024),
						"contentType": "image/png",
					},
				},
			},
		},
	}
}

func TestMediaUploadWebhookHandler_ObjectCreated_Success(t *testing.T) {
	process := &mockProcessHandler{}
	h := handler.NewMediaUploadWebhookHandler(testMediaBucket, process)
	app := testutil.NewTestApp()
	app.Post("/webhooks/minio/object-created", h.ObjectCreated)

	key := "clients/" + testutil.TestClientID.String() + "/campaigns/" +
		testutil.TestCampaignID.String() + "/abc.png"
	resp, err := app.Test(mustJSONRequest(t, objectCreatedPayload(testMediaBucket, key)))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	process.waitCalled(t)
	process.mu.Lock()
	defer process.mu.Unlock()
	if len(process.calls) != 1 || process.calls[0].ObjectKey != key {
		t.Fatalf("unexpected calls: %+v", process.calls)
	}
}

func TestMediaUploadWebhookHandler_ObjectCreated_UnknownKeyStillOK(t *testing.T) {
	process := &mockProcessHandler{}
	h := handler.NewMediaUploadWebhookHandler(testMediaBucket, process)
	app := testutil.NewTestApp()
	app.Post("/webhooks/minio/object-created", h.ObjectCreated)

	resp, err := app.Test(mustJSONRequest(t, objectCreatedPayload(testMediaBucket, "other/path.png")))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	process.waitCalled(t)
}

func TestMediaUploadWebhookHandler_ObjectCreated_SkipsThumbnailKey(t *testing.T) {
	process := &mockProcessHandler{}
	h := handler.NewMediaUploadWebhookHandler(testMediaBucket, process)
	app := testutil.NewTestApp()
	app.Post("/webhooks/minio/object-created", h.ObjectCreated)

	key := "clients/" + testutil.TestClientID.String() + "/campaigns/" +
		testutil.TestCampaignID.String() + "/thumbnails/" + testutil.TestCampaignID.String() + ".jpg"
	resp, err := app.Test(mustJSONRequest(t, objectCreatedPayload(testMediaBucket, key)))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	time.Sleep(50 * time.Millisecond)
	process.mu.Lock()
	defer process.mu.Unlock()
	if len(process.calls) != 0 {
		t.Fatalf("expected thumbnail key to be skipped, got %+v", process.calls)
	}
}

func TestMediaUploadWebhookHandler_ObjectCreated_InvalidPayload(t *testing.T) {
	h := handler.NewMediaUploadWebhookHandler(testMediaBucket, &mockProcessHandler{})
	app := testutil.NewTestApp()
	app.Post("/webhooks/minio/object-created", h.ObjectCreated)

	req, err := http.NewRequest(http.MethodPost, "/webhooks/minio/object-created", bytes.NewReader([]byte("{")))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaUploadWebhookHandler_ObjectCreated_ValidationFailed(t *testing.T) {
	h := handler.NewMediaUploadWebhookHandler(testMediaBucket, &mockProcessHandler{})
	app := testutil.NewTestApp()
	app.Post("/webhooks/minio/object-created", h.ObjectCreated)

	resp, err := app.Test(mustJSONRequest(t, map[string]any{"Records": []any{}}))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaUploadWebhookMiddleware_Unauthorized(t *testing.T) {
	mw := middleware.NewMediaUploadWebhookMiddleware("secret")
	h := handler.NewMediaUploadWebhookHandler(testMediaBucket, &mockProcessHandler{})
	app := testutil.NewTestApp()
	app.Post("/webhooks/minio/object-created", mw.Protected(), h.ObjectCreated)

	req, err := testutil.JSONRequest(
		http.MethodPost,
		"/webhooks/minio/object-created",
		objectCreatedPayload(testMediaBucket, "x.png"),
	)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer wrong")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
}

func TestMediaUploadWebhookMiddleware_Authorized(t *testing.T) {
	mw := middleware.NewMediaUploadWebhookMiddleware("secret")
	process := &mockProcessHandler{}
	h := handler.NewMediaUploadWebhookHandler(testMediaBucket, process)
	app := testutil.NewTestApp()
	app.Post("/webhooks/minio/object-created", mw.Protected(), h.ObjectCreated)

	req, err := testutil.JSONRequest(
		http.MethodPost,
		"/webhooks/minio/object-created",
		objectCreatedPayload(testMediaBucket, "x.png"),
	)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer secret")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	process.waitCalled(t)
}

func TestMediaUploadWebhookHandler_ObjectCreated_WrongBucket(t *testing.T) {
	process := &mockProcessHandler{}
	h := handler.NewMediaUploadWebhookHandler(testMediaBucket, process)
	app := testutil.NewTestApp()
	app.Post("/webhooks/minio/object-created", h.ObjectCreated)

	resp, err := app.Test(mustJSONRequest(t, objectCreatedPayload("other-bucket", "x.png")))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	time.Sleep(50 * time.Millisecond)
	process.mu.Lock()
	defer process.mu.Unlock()
	if len(process.calls) != 0 {
		t.Fatalf("expected wrong bucket to be skipped, got %+v", process.calls)
	}
}

func TestMediaUploadWebhookHandler_ObjectCreated_InvalidObjectKey(t *testing.T) {
	process := &mockProcessHandler{}
	h := handler.NewMediaUploadWebhookHandler(testMediaBucket, process)
	app := testutil.NewTestApp()
	app.Post("/webhooks/minio/object-created", h.ObjectCreated)

	resp, err := app.Test(mustJSONRequest(t, objectCreatedPayload(testMediaBucket, "%zz")))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	time.Sleep(50 * time.Millisecond)
	process.mu.Lock()
	defer process.mu.Unlock()
	if len(process.calls) != 0 {
		t.Fatalf("expected invalid key to be skipped, got %+v", process.calls)
	}
}

func TestMediaUploadWebhookHandler_ObjectCreated_ProcessError(t *testing.T) {
	process := &mockProcessHandler{err: errProcessFailed}
	h := handler.NewMediaUploadWebhookHandler(testMediaBucket, process)
	app := testutil.NewTestApp()
	app.Post("/webhooks/minio/object-created", h.ObjectCreated)

	resp, err := app.Test(mustJSONRequest(t, objectCreatedPayload(testMediaBucket, "x.png")))
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	process.waitCalled(t)
}

var errProcessFailed = errors.New("process failed")

func mustJSONRequest(t *testing.T, body any) *http.Request {
	t.Helper()
	req, err := testutil.JSONRequest(http.MethodPost, "/webhooks/minio/object-created", body)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	return req
}
