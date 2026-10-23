package services

import (
	"github.com/google/uuid"
	"your-project/internal/models"
	"your-project/internal/repository"
)

type RatingService struct {
	repo *repository.RatingRepository
}

func NewRatingService(repo *repository.RatingRepository) *RatingService {
	return &RatingService{repo: repo}
}

func (s *RatingService) CreateRating(productID uuid.UUID, score float64) error {
	if score < 1 || score > 5 {
		return fmt.Errorf("score must be between 1 and 5")
	}
	return s.repo.Create(productID, score)
}

func (s *RatingService) GetRatingsByProduct(productID uuid.UUID) ([]models.ProductRating, error) {
	return s.repo.GetByProductID(productID)
}

func (s *RatingService) GetRatingSummary(productID uuid.UUID) (map[string]interface{}, error) {
	avg, count, err := s.repo.GetSummary(productID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"average_rating": avg,
		"total_ratings":  count,
	}, nil
}