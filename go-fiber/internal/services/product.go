package services

import (
	"database/sql"

	"github.com/google/uuid"
	"your-project/internal/models"
	"your-project/internal/repository"
)

type ProductService struct {
	repo *repository.ProductRepository
}

func NewProductService(repo *repository.ProductRepository) *ProductService {
	return &ProductService{repo: repo}
}

func (s *ProductService) CreateProduct(product *models.Product) error {
	return s.repo.Create(product)
}

func (s *ProductService) GetProduct(id uuid.UUID) (*models.Product, error) {
	return s.repo.GetByID(id)
}

func (s *ProductService) ListProducts() ([]models.Product, error) {
	return s.repo.List()
}

func (s *ProductService) UpdateProduct(id uuid.UUID, product *models.Product) error {
	// Check if product exists
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	// Update only non-zero fields
	if product.Name != "" {
		existing.Name = product.Name
	}
	if product.Price > 0 {
		existing.Price = product.Price
	}
	if product.Quantity >= 0 {
		existing.Quantity = product.Quantity
	}
	if len(product.Metadata) > 0 {
		existing.Metadata = product.Metadata
	}
	if len(product.RelatedIDs) > 0 {
		existing.RelatedIDs = product.RelatedIDs
	}
	if len(product.Categories) > 0 {
		existing.Categories = product.Categories
	}

	return s.repo.Update(id, existing)
}

func (s *ProductService) DeleteProduct(id uuid.UUID) error {
	return s.repo.Delete(id)
}

func (s *ProductService) BulkCreateProducts(products []models.Product) error {
	return s.repo.BulkCreate(products)
}