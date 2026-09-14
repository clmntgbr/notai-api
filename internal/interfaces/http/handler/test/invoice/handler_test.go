package invoicetest

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	queryclient "go-api/internal/application/query/client"
	queryinvoice "go-api/internal/application/query/invoice"
	domainclient "go-api/internal/domain/client"
	domaininvoice "go-api/internal/domain/invoice"
	"go-api/internal/domain/paginate"
	"go-api/internal/interfaces/http/handler"
	"go-api/internal/interfaces/http/testutil"

	"github.com/google/uuid"
)

type mockListInvoicesHandler struct {
	called   bool
	query    queryinvoice.ListInvoicesQuery
	invoices []*domaininvoice.InvoiceView
	total    int64
	err      error
}

func (m *mockListInvoicesHandler) Handle(
	_ context.Context,
	q queryinvoice.ListInvoicesQuery,
) ([]*domaininvoice.InvoiceView, int64, error) {
	m.called = true
	m.query = q
	return m.invoices, m.total, m.err
}

type mockGetClientByIDHandler struct {
	view    *domainclient.ClientView
	err     error
	nilView bool
}

func (m *mockGetClientByIDHandler) Handle(
	_ context.Context,
	_ queryclient.GetClientByIDQuery,
) (*domainclient.ClientView, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.nilView {
		return nil, nil
	}
	if m.view != nil {
		return m.view, nil
	}
	return &domainclient.ClientView{
		ID:          testutil.TestClientID,
		Name:        "Test Client",
		WorkspaceID: testutil.TestWorkspaceID,
		MemberIDs:   []uuid.UUID{testutil.TestUserID},
	}, nil
}

func sampleInvoice() *domaininvoice.InvoiceView {
	return &domaininvoice.InvoiceView{
		ID:              uuid.MustParse("01960000-0000-7000-8000-000000000020"),
		WorkspaceID:     testutil.TestWorkspaceID,
		StripeInvoiceID: "in_123",
		Number:          "INV-1",
		Status:          "paid",
		Currency:        "eur",
		Total:           2900,
		StripeCreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func newInvoiceHandler(list *mockListInvoicesHandler, getClient *mockGetClientByIDHandler) *handler.InvoiceHandler {
	if list == nil {
		list = &mockListInvoicesHandler{}
	}
	if getClient == nil {
		getClient = &mockGetClientByIDHandler{}
	}
	return handler.NewInvoiceHandler(getClient, list)
}

func TestInvoiceHandler_GetInvoices_Success(t *testing.T) {
	list := &mockListInvoicesHandler{
		invoices: []*domaininvoice.InvoiceView{sampleInvoice()},
		total:    1,
	}
	h := newInvoiceHandler(list, nil)

	app := testutil.NewTestApp()
	app.Get("/invoices", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetInvoices)

	req, err := testutil.JSONRequest(http.MethodGet, "/invoices", nil)
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
	if !list.called || list.query.WorkspaceID != testutil.TestWorkspaceID {
		t.Fatalf("unexpected query: %+v", list.query)
	}
	if list.query.Query.SortBy != "stripe_created_at" || list.query.Query.OrderBy != paginate.OrderByDesc {
		t.Fatalf("unexpected pagination defaults: %+v", list.query.Query)
	}

	var out paginate.PaginateResponse
	testutil.DecodeJSON(t, resp, &out)
	if out.Total != 1 {
		t.Fatalf("total: got %d want 1", out.Total)
	}
}

func TestInvoiceHandler_GetInvoices_Unauthorized(t *testing.T) {
	h := newInvoiceHandler(nil, nil)
	app := testutil.NewTestApp()
	app.Get("/invoices", h.GetInvoices)

	req, _ := testutil.JSONRequest(http.MethodGet, "/invoices", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestInvoiceHandler_GetInvoices_MissingActiveClient(t *testing.T) {
	h := newInvoiceHandler(nil, nil)
	app := testutil.NewTestApp()
	app.Get("/invoices", testutil.WithUserWithoutClient(testutil.TestUserID), h.GetInvoices)

	req, _ := testutil.JSONRequest(http.MethodGet, "/invoices", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestInvoiceHandler_GetInvoices_HandlerError_Internal(t *testing.T) {
	list := &mockListInvoicesHandler{err: errors.New("db down")}
	h := newInvoiceHandler(list, nil)
	app := testutil.NewTestApp()
	app.Get("/invoices", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetInvoices)

	req, _ := testutil.JSONRequest(http.MethodGet, "/invoices", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

func TestInvoiceHandler_GetInvoices_WithSortDefaults(t *testing.T) {
	list := &mockListInvoicesHandler{invoices: []*domaininvoice.InvoiceView{}, total: 0}
	h := newInvoiceHandler(list, nil)
	app := testutil.NewTestApp()
	app.Get("/invoices", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetInvoices)

	req, err := testutil.JSONRequest(http.MethodGet, "/invoices?orderBy=asc&sortBy=created_at", nil)
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
	if list.query.Query.SortBy != "created_at" || list.query.Query.OrderBy != paginate.OrderByAsc {
		t.Fatalf("unexpected query: %+v", list.query.Query)
	}
}

func TestInvoiceHandler_GetInvoices_ClientNotFound(t *testing.T) {
	getClient := &mockGetClientByIDHandler{err: errors.New("client not found")}
	h := newInvoiceHandler(nil, getClient)
	app := testutil.NewTestApp()
	app.Get("/invoices", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetInvoices)

	req, _ := testutil.JSONRequest(http.MethodGet, "/invoices", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestInvoiceHandler_GetInvoices_ClientNil(t *testing.T) {
	getClient := &mockGetClientByIDHandler{nilView: true}
	h := newInvoiceHandler(nil, getClient)
	app := testutil.NewTestApp()
	app.Get("/invoices", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetInvoices)

	req, _ := testutil.JSONRequest(http.MethodGet, "/invoices", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestInvoiceHandler_GetInvoices_ResolveClientError(t *testing.T) {
	getClient := &mockGetClientByIDHandler{err: errors.New("db")}
	h := newInvoiceHandler(nil, getClient)
	app := testutil.NewTestApp()
	app.Get("/invoices", testutil.WithActiveClient(testutil.TestUserID, testutil.TestClientID), h.GetInvoices)

	req, _ := testutil.JSONRequest(http.MethodGet, "/invoices", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}
