package proxy_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yuya-cpu/web-security-proxy/internal/proxy"
)

func TestProxy_SaveErrorTransactionOnUpstreamFailure(t *testing.T) {
	recorder := &memoryRecorder{}
	proxyHandler := proxy.NewServer(recorder)

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:1/unreachable", nil)
	req.URL.Host = "127.0.0.1:1"
	req.Host = "127.0.0.1:1"
	rec := httptest.NewRecorder()

	proxyHandler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadGateway, rec.Code)
	recorded := recorder.last()
	require.NotNil(t, recorded)
	assert.Equal(t, http.StatusBadGateway, recorded.StatusCode)
	assert.Contains(t, recorded.ResponseBody, "127.0.0.1:1")
}

func TestProxy_SkipsSaveWhenRecorderNil(t *testing.T) {
	proxyHandler := proxy.NewServer(nil)
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:1/unreachable", nil)
	req.URL.Host = "127.0.0.1:1"
	req.Host = "127.0.0.1:1"
	rec := httptest.NewRecorder()

	proxyHandler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadGateway, rec.Code)
}

func TestProxy_RecorderReceivesContext(t *testing.T) {
	recorder := &memoryRecorder{}
	proxyHandler := proxy.NewServer(recorder)

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:1/unreachable", nil)
	req = req.WithContext(context.WithValue(req.Context(), struct{}{}, "test"))
	req.URL.Host = "127.0.0.1:1"
	req.Host = "127.0.0.1:1"
	rec := httptest.NewRecorder()

	proxyHandler.ServeHTTP(rec, req)
	require.NotNil(t, recorder.last())
}
