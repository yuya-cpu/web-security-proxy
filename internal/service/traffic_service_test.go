package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yuya-cpu/web-security-proxy/internal/model"
	"github.com/yuya-cpu/web-security-proxy/internal/service"
)

type mockRepository struct {
	saved []*model.HTTPTransaction
	items []model.HTTPTransaction
}

func (m *mockRepository) Save(ctx context.Context, tx *model.HTTPTransaction) (int64, error) {
	m.saved = append(m.saved, tx)
	id := int64(len(m.saved))
	tx.ID = id
	m.items = append([]model.HTTPTransaction{*tx}, m.items...)
	return id, nil
}

func (m *mockRepository) List(ctx context.Context, limit int) ([]model.HTTPTransaction, error) {
	if limit > len(m.items) {
		return m.items, nil
	}
	return m.items[:limit], nil
}

func (m *mockRepository) GetByID(ctx context.Context, id int64) (*model.HTTPTransaction, error) {
	for _, item := range m.items {
		if item.ID == id {
			copyItem := item
			return &copyItem, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func TestTrafficService_SaveTransaction(t *testing.T) {
	repo := &mockRepository{}
	svc := service.NewTrafficService(repo)

	id, err := svc.SaveTransaction(context.Background(), &model.HTTPTransaction{
		Method: "POST",
		URL:    "http://example.com/api",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), id)
}

func TestTrafficService_SaveTransaction_Validation(t *testing.T) {
	svc := service.NewTrafficService(&mockRepository{})

	_, err := svc.SaveTransaction(context.Background(), nil)
	require.Error(t, err)

	_, err = svc.SaveTransaction(context.Background(), &model.HTTPTransaction{URL: "http://example.com"})
	require.Error(t, err)

	_, err = svc.SaveTransaction(context.Background(), &model.HTTPTransaction{Method: "GET"})
	require.Error(t, err)
}

func TestTrafficService_GetTransaction(t *testing.T) {
	repo := &mockRepository{
		items: []model.HTTPTransaction{{ID: 1, Method: "GET", URL: "http://example.com"}},
	}
	svc := service.NewTrafficService(repo)

	tx, err := svc.GetTransaction(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "GET", tx.Method)
}

func TestTrafficService_GetTransaction_InvalidID(t *testing.T) {
	svc := service.NewTrafficService(&mockRepository{})
	_, err := svc.GetTransaction(context.Background(), 0)
	require.Error(t, err)
}
