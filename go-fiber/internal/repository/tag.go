package repository

import (
	"database/sql"

	"github.com/google/uuid"
	"your-project/internal/models"
)

type TagRepository struct {
	db *sql.DB
}

func NewTagRepository(db *sql.DB) *TagRepository {
	return &TagRepository{db: db}
}

func (r *TagRepository) AddTags(productID uuid.UUID, tags []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO product_tags (product_id, tag)
		VALUES ($1, $2)
		ON CONFLICT (product_id, tag) DO NOTHING
	`
	
	stmt, err := tx.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, tag := range tags {
		_, err := stmt.Exec(productID, tag)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *TagRepository) GetByProductID(productID uuid.UUID) ([]string, error) {
	query := `
		SELECT tag FROM product_tags
		WHERE product_id = $1
		ORDER BY created_at DESC
	`
	
	rows, err := r.db.Query(query, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			continue
		}
		tags = append(tags, tag)
	}

	return tags, nil
}

func (r *TagRepository) GetProductsByTag(tag string) ([]models.Product, error) {
	query := `
		SELECT p.id, p.name, p.price, p.quantity, p.metadata, p.related_ids, p.categories, p.created_at, p.updated_at
		FROM products p
		INNER JOIN product_tags pt ON p.id = pt.product_id
		WHERE pt.tag = $1
		ORDER BY p.created_at DESC
	`
	
	rows, err := r.db.Query(query, tag)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var product models.Product
		var metadataJSON, relatedJSON, categoriesJSON []byte

		err := rows.Scan(
			&product.ID, &product.Name, &product.Price, &product.Quantity,
			&metadataJSON, &relatedJSON, &categoriesJSON,
			&product.CreatedAt, &product.UpdatedAt,
		)
		if err != nil {
			continue
		}

		// Unmarshal JSON fields (implement similar to product repository)
		products = append(products, product)
	}

	return products, nil
}