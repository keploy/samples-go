package services

import (
	"github.com/google/uuid"
	"your-project/internal/models"
	"your-project/internal/repository"
)

type TagService struct {
	repo *repository.TagRepository
}

func NewTagService(repo *repository.TagRepository) *TagService {
	return &TagService{repo: repo}
}

func (s *TagService) AddTags(productID uuid.UUID, tags []string) error {
	return s.repo.AddTags(productID, tags)
}

func (s *TagService) GetTagsByProduct(productID uuid.UUID) ([]string, error) {
	return s.repo.GetByProductID(productID)
}

func (s *TagService) GetProductsByTag(tag string) ([]models.Product, error) {
	return s.repo.GetProductsByTag(tag)
}