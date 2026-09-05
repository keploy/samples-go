package repository

import (
	"database/sql"

	"github.com/google/uuid"
	"your-project/internal/models"
)

type RatingRepository struct {
	db *sql.DB
}

func NewRatingRepository(db *sql.DB) *RatingRepository {
	return &RatingRepository{db: db}
}

func (r *RatingRepository) Create(productID uuid.UUID, score float64) error {
	query := `
		INSERT INTO product_ratings (product_id, score)
		VALUES ($1, $2)
	`
	_, err := r.db.Exec(query, productID, score)
	return err
}

func (r *RatingRepository) GetByProductID(productID uuid.UUID) ([]models.ProductRating, error) {
	query := `
		SELECT id, product_id, score, created_at
		FROM product_ratings
		WHERE product_id = $1
		ORDER BY created_at DESC
		LIMIT 10
	`
	
	rows, err := r.db.Query(query, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ratings []models.ProductRating
	for rows.Next() {
		var rating models.ProductRating
		err := rows.Scan(&rating.ID, &rating.ProductID, &rating.Score, &rating.CreatedAt)
		if err != nil {
			continue
		}
		ratings = append(ratings, rating)
	}

	return ratings, nil
}

func (r *RatingRepository) GetSummary(productID uuid.UUID) (float64, int, error) {
	query := `
		SELECT COALESCE(AVG(score), 0), COUNT(*)
		FROM product_ratings
		WHERE product_id = $1
	`
	
	var avg float64
	var count int
	err := r.db.QueryRow(query, productID).Scan(&avg, &count)
	return avg, count, err
}