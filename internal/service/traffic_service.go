package service

import (
	"context"
	"fmt"

	"github.com/yuya-cpu/web-security-proxy/internal/model"
	"github.com/yuya-cpu/web-security-proxy/internal/repository"
)

//
type TrafficService struct {
	repo repository.TrafficRepository
}

func NewTrafficService(repo repository.TrafficRepository) *TrafficService {
	return &TrafficService{repo: repo}
}

func (s *TrafficService) SaveTransaction(ctx context.Context, tx *model.HTTPTransaction) (int64, error) {
	if tx == nil {
		return 0, fmt.Errorf("transaction is nil")
	}
	if tx.Method == "" {
		return 0, fmt.Errorf("method is required")
	}
	if tx.URL == "" {
		return 0, fmt.Errorf("url is required")
	}

	return s.repo.Save(ctx, tx)
}

func (s *TrafficService) ListTransactions(ctx context.Context, limit int) ([]model.HTTPTransaction, error) {
	return s.repo.List(ctx, limit)
}

func (s *TrafficService) GetTransaction(ctx context.Context, id int64) (*model.HTTPTransaction, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid id: %d", id)
	}
	return s.repo.GetByID(ctx, id)
}
