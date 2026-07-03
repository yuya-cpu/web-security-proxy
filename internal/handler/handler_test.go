package handler_test

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yuya-cpu/web-security-proxy/internal/handler"
	"github.com/yuya-cpu/web-security-proxy/internal/model"
	"github.com/yuya-cpu/web-security-proxy/internal/scanner"
	"github.com/yuya-cpu/web-security-proxy/internal/service"
)

type mockRepo struct {
	items []model.HTTPTransaction
}

func (m *mockRepo) Save(_ context.Context, tx *model.HTTPTransaction) (int64, error) {
	tx.ID = int64(len(m.items) + 1)
	m.items = append([]model.HTTPTransaction{*tx}, m.items...)
	return tx.ID, nil
}

func (m *mockRepo) List(_ context.Context, limit int) ([]model.HTTPTransaction, error) {
	if limit > len(m.items) {
		return m.items, nil
	}
	return m.items[:limit], nil
}

func (m *mockRepo) GetByID(_ context.Context, id int64) (*model.HTTPTransaction, error) {
	for _, item := range m.items {
		if item.ID == id {
			copyItem := item
			return &copyItem, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func newTestHandler(t *testing.T, repo *mockRepo, client *http.Client) http.Handler {
	t.Helper()
	svc := service.NewTrafficService(repo)
	repeaterSvc := service.NewRepeaterService(repo, client)
	diagnosticSvc := service.NewDiagnosticService(scanner.NewDiagnosticScanner())
	scanSvc := service.NewScanService(scanner.NewActiveScanner(nil))
	tmpl := template.Must(template.New("index.html").Parse(`{{define "index.html"}}{{.Title}}{{end}}`))
	h := handler.NewHandler(svc, repeaterSvc, diagnosticSvc, scanSvc, tmpl)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux
}

func TestHandler_APIListTransactions(t *testing.T) {
	repo := &mockRepo{
		items: []model.HTTPTransaction{{ID: 1, Method: "GET", URL: "http://example.com", StatusCode: 200}},
	}
	mux := newTestHandler(t, repo, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/transactions", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"method":"GET"`)
}

func TestHandler_APIGetTransaction(t *testing.T) {
	repo := &mockRepo{
		items: []model.HTTPTransaction{{ID: 1, Method: "GET", URL: "http://example.com", StatusCode: 200}},
	}
	mux := newTestHandler(t, repo, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/transactions/1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"status_code":200`)
}

func TestHandler_APIGetTransaction_NotFound(t *testing.T) {
	mux := newTestHandler(t, &mockRepo{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/transactions/99", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_Index(t *testing.T) {
	repo := &mockRepo{
		items: []model.HTTPTransaction{{ID: 1, Method: "GET", URL: "http://example.com", StatusCode: 200}},
	}
	mux := newTestHandler(t, repo, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Traffic History")
}

func TestHandler_Detail(t *testing.T) {
	repo := &mockRepo{
		items: []model.HTTPTransaction{{ID: 1, Method: "GET", URL: "http://example.com", StatusCode: 200}},
	}
	svc := service.NewTrafficService(repo)
	repeaterSvc := service.NewRepeaterService(repo, nil)
	diagnosticSvc := service.NewDiagnosticService(scanner.NewDiagnosticScanner())
	scanSvc := service.NewScanService(scanner.NewActiveScanner(nil))
	tmpl := template.Must(template.ParseFiles("../../web/templates/index.html"))
	h := handler.NewHandler(svc, repeaterSvc, diagnosticSvc, scanSvc, tmpl)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/transactions/1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "http://example.com")
	assert.Contains(t, rec.Body.String(), "Repeater")
	assert.Contains(t, rec.Body.String(), "Security")
	assert.Contains(t, rec.Body.String(), "Scanner")
}

func TestHandler_APIActiveScan_RejectsCONNECT(t *testing.T) {
	repo := &mockRepo{
		items: []model.HTTPTransaction{{
			ID:     1,
			Method: "CONNECT",
			URL:    "https://example.com:443",
		}},
	}
	mux := newTestHandler(t, repo, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/transactions/1/active-scan", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_APIDiagnostics(t *testing.T) {
	repo := &mockRepo{
		items: []model.HTTPTransaction{{
			ID:              1,
			Method:          "GET",
			URL:             "https://example.com",
			ResponseHeaders: "Server: apache\nSet-Cookie: sid=1; Path=/",
			StatusCode:      200,
		}},
	}
	mux := newTestHandler(t, repo, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/transactions/1/diagnostics", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"server":"apache"`)
	assert.Contains(t, rec.Body.String(), `"findings"`)
	assert.Contains(t, rec.Body.String(), `"cookies"`)
}

func TestHandler_APIRepeaterSend(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	repo := &mockRepo{}
	mux := newTestHandler(t, repo, upstream.Client())

	body := `{"method":"GET","url":"` + upstream.URL + `","headers":"","body":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/repeater/send", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), `"status_code":200`)
	assert.Len(t, repo.items, 1)
}

func TestHandler_APIRepeaterSend_InvalidJSON(t *testing.T) {
	mux := newTestHandler(t, &mockRepo{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/repeater/send", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
