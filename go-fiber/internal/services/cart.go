package services

import (
	"context"

	"your-project/internal/repository"
)

type CartService struct {
	repo *repository.CartRepository
}

func NewCartService(repo *repository.CartRepository) *CartService {
	return &CartService{repo: repo}
}

func (s *CartService) UpdateCart(ctx context.Context, userID string, cart map[string]int) error {
	return s.repo.Update(ctx, userID, cart)
}

func (s *CartService) GetCart(ctx context.Context, userID string) (map[string]int, error) {
	return s.repo.Get(ctx, userID)
}