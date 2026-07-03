package service_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yuya-cpu/web-security-proxy/internal/model"
	"github.com/yuya-cpu/web-security-proxy/internal/service"
)

type repeaterMockRepo struct {
	saved []*model.HTTPTransaction
}

func (m *repeaterMockRepo) Save(_ context.Context, tx *model.HTTPTransaction) (int64, error) {
	m.saved = append(m.saved, tx)
	id := int64(len(m.saved))
	tx.ID = id
	return id, nil
}

func (m *repeaterMockRepo) List(_ context.Context, _ int) ([]model.HTTPTransaction, error) {
	return nil, nil
}

func (m *repeaterMockRepo) GetByID(_ context.Context, _ int64) (*model.HTTPTransaction, error) {
	return nil, nil
}

func TestRepeaterService_Send_Success(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		body, _ := io.ReadAll(r.Body)
		assert.Equal(t, `{"name":"test"}`, string(body))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.Header().Set("X-Test", "ok")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":1}`))
	}))
	defer upstream.Close()

	repo := &repeaterMockRepo{}
	svc := service.NewRepeaterService(repo, upstream.Client())

	tx, err := svc.Send(context.Background(), model.RepeaterRequest{
		Method:  "POST",
		URL:     upstream.URL + "/api/items",
		Headers: "Content-Type: application/json",
		Body:    `{"name":"test"}`,
	})
	require.NoError(t, err)
	require.NotNil(t, tx)

	assert.Equal(t, int64(1), tx.ID)
	assert.Equal(t, http.StatusCreated, tx.StatusCode)
	assert.Contains(t, tx.ResponseBody, `"id":1`)
	assert.Len(t, repo.saved, 1)
}

func TestRepeaterService_Send_Validation(t *testing.T) {
	svc := service.NewRepeaterService(&repeaterMockRepo{}, nil)

	_, err := svc.Send(context.Background(), model.RepeaterRequest{
		URL: "http://example.com",
	})
	require.Error(t, err)

	_, err = svc.Send(context.Background(), model.RepeaterRequest{
		Method: "GET",
	})
	require.Error(t, err)

	_, err = svc.Send(context.Background(), model.RepeaterRequest{
		Method: "CONNECT",
		URL:    "https://example.com:443",
	})
	require.Error(t, err)
}

func TestRepeaterService_Send_InvalidHeader(t *testing.T) {
	svc := service.NewRepeaterService(&repeaterMockRepo{}, nil)

	_, err := svc.Send(context.Background(), model.RepeaterRequest{
		Method:  "GET",
		URL:     "http://example.com",
		Headers: "invalid-header-line",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid header line")
}

func TestRepeaterService_Send_UpstreamErrorStillSaved(t *testing.T) {
	repo := &repeaterMockRepo{}
	svc := service.NewRepeaterService(repo, &http.Client{Timeout: 0})

	tx, err := svc.Send(context.Background(), model.RepeaterRequest{
		Method: "GET",
		URL:    "http://127.0.0.1:1/unreachable",
	})
	require.NoError(t, err)
	require.NotNil(t, tx)

	assert.Equal(t, http.StatusBadGateway, tx.StatusCode)
	assert.Contains(t, tx.ResponseBody, "127.0.0.1:1")
	assert.Len(t, repo.saved, 1)
}

func TestRepeaterService_Send_PreservesMultipleHeaders(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "value1", r.Header.Get("X-Custom"))
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	repo := &repeaterMockRepo{}
	svc := service.NewRepeaterService(repo, upstream.Client())

	_, err := svc.Send(context.Background(), model.RepeaterRequest{
		Method:  "GET",
		URL:     upstream.URL,
		Headers: "X-Custom: value1\nAccept: text/plain",
		Body:    "",
	})
	require.NoError(t, err)
	assert.True(t, strings.Contains(repo.saved[0].RequestHeaders, "X-Custom"))
}
