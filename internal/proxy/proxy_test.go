package proxy_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yuya-cpu/web-security-proxy/internal/model"
	"github.com/yuya-cpu/web-security-proxy/internal/proxy"
)

type memoryRecorder struct {
	mu    sync.Mutex
	items []*model.HTTPTransaction
}

func (m *memoryRecorder) SaveTransaction(ctx context.Context, tx *model.HTTPTransaction) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = append(m.items, tx)
	return int64(len(m.items)), nil
}

func (m *memoryRecorder) last() *model.HTTPTransaction {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.items) == 0 {
		return nil
	}
	return m.items[len(m.items)-1]
}

func TestProxy_ForwardsHTTPAndRecordsTransaction(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		body, _ := io.ReadAll(r.Body)
		assert.Equal(t, "hello", string(body))
		w.Header().Set("X-Test", "ok")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("created"))
	}))
	defer upstream.Close()

	recorder := &memoryRecorder{}
	proxyServer := httptest.NewServer(proxy.NewServer(recorder))
	defer proxyServer.Close()

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(mustParseURL(t, proxyServer.URL)),
		},
	}

	req, err := http.NewRequest(http.MethodPost, upstream.URL+"/items", strings.NewReader("hello"))
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "created", string(body))

	recorded := recorder.last()
	require.NotNil(t, recorded)
	assert.Equal(t, "POST", recorded.Method)
	assert.Contains(t, recorded.URL, "/items")
	assert.Equal(t, http.StatusCreated, recorded.StatusCode)
	assert.Contains(t, recorded.ResponseBody, "created")
}

func TestProxy_CONNECT_IsHandled(t *testing.T) {
	recorder := &memoryRecorder{}
	proxyHandler := proxy.NewServer(recorder)

	req := httptest.NewRequest(http.MethodConnect, "http://127.0.0.1:1", nil)
	req.Host = "127.0.0.1:1"
	rec := httptest.NewRecorder()

	proxyHandler.ServeHTTP(rec, req)

	recorded := recorder.last()
	require.NotNil(t, recorded)
	assert.Equal(t, http.MethodConnect, recorded.Method)
	assert.Equal(t, "https://127.0.0.1:1", recorded.URL)
}

func TestFormatHeadersAndDumpRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	req.Header.Set("User-Agent", "test")

	dump, err := proxy.DumpRequest(req)
	require.NoError(t, err)
	assert.Contains(t, dump, "GET")
	assert.Contains(t, dump, "User-Agent")
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	return parsed
}
