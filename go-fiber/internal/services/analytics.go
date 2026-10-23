package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"your-project/internal/models"
	"your-project/internal/repository"
)

type AnalyticsService struct {
	cartRepo    *repository.CartRepository
	productRepo *repository.ProductRepository
}

func NewAnalyticsService(cartRepo *repository.CartRepository, productRepo *repository.ProductRepository) *AnalyticsService {
	return &AnalyticsService{
		cartRepo:    cartRepo,
		productRepo: productRepo,
	}
}

func (s *AnalyticsService) LogActivity(ctx context.Context, message string) error {
	return s.cartRepo.LogActivity(ctx, message)
}

func (s *AnalyticsService) GetActivityLog(ctx context.Context, limit int64) ([]string, error) {
	return s.cartRepo.GetActivityLog(ctx, limit)
}

func (s *AnalyticsService) TrackVisitor(ctx context.Context, productID, visitorID string) error {
	return s.cartRepo.IncrementVisitor(ctx, productID, visitorID)
}

func (s *AnalyticsService) GetVisitorCount(ctx context.Context, productID string) (int64, error) {
	return s.cartRepo.GetVisitorCount(ctx, productID)
}

func (s *AnalyticsService) UpdateLeaderboard(ctx context.Context, productID string, sales float64) error {
	return s.cartRepo.UpdateLeaderboard(ctx, productID, sales)
}

func (s *AnalyticsService) GetLeaderboard(ctx context.Context, limit int64) ([]map[string]interface{}, error) {
	results, err := s.cartRepo.GetLeaderboard(ctx, limit)
	if err != nil {
		return nil, err
	}

	var leaderboard []map[string]interface{}
	for _, result := range results {
		productID, err := uuid.Parse(result.Member.(string))
		if err != nil {
			continue
		}

		product, err := s.productRepo.GetByID(productID)
		if err != nil {
			continue
		}

		leaderboard = append(leaderboard, map[string]interface{}{
			"product": product,
			"sales":   result.Score,
		})
	}

	return leaderboard, nil
}