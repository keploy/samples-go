package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"your-project/internal/models"
)

type ProductRepository struct {
	db *sql.DB
}

func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) Create(product *models.Product) error {
	product.ID = uuid.New()
	
	metadataJSON, _ := json.Marshal(product.Metadata)
	relatedJSON, _ := json.Marshal(product.RelatedIDs)
	categoriesJSON, _ := json.Marshal(product.Categories)

	query := `
		INSERT INTO products (id, name, price, quantity, metadata, related_ids, categories)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at
	`
	
	return r.db.QueryRow(query, product.ID, product.Name, product.Price, 
		product.Quantity, metadataJSON, relatedJSON, categoriesJSON).
		Scan(&product.CreatedAt, &product.UpdatedAt)
}

func (r *ProductRepository) GetByID(id uuid.UUID) (*models.Product, error) {
	product := &models.Product{}
	var metadataJSON, relatedJSON, categoriesJSON []byte

	query := `
		SELECT id, name, price, quantity, metadata, related_ids, categories, created_at, updated_at
		FROM products WHERE id = $1
	`
	
	err := r.db.QueryRow(query, id).Scan(
		&product.ID, &product.Name, &product.Price, &product.Quantity,
		&metadataJSON, &relatedJSON, &categoriesJSON,
		&product.CreatedAt, &product.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}

	json.Unmarshal(metadataJSON, &product.Metadata)
	json.Unmarshal(relatedJSON, &product.RelatedIDs)
	json.Unmarshal(categoriesJSON, &product.Categories)

	return product, nil
}

func (r *ProductRepository) List() ([]models.Product, error) {
	query := `
		SELECT id, name, price, quantity, metadata, related_ids, categories, created_at, updated_at
		FROM products ORDER BY created_at DESC
	`
	
	rows, err := r.db.Query(query)
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

		json.Unmarshal(metadataJSON, &product.Metadata)
		json.Unmarshal(relatedJSON, &product.RelatedIDs)
		json.Unmarshal(categoriesJSON, &product.Categories)

		products = append(products, product)
	}

	return products, nil
}

func (r *ProductRepository) Update(id uuid.UUID, product *models.Product) error {
	metadataJSON, _ := json.Marshal(product.Metadata)
	relatedJSON, _ := json.Marshal(product.RelatedIDs)
	categoriesJSON, _ := json.Marshal(product.Categories)

	query := `
		UPDATE products 
		SET name = $2, price = $3, quantity = $4, metadata = $5, 
		    related_ids = $6, categories = $7, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`
	
	return r.db.QueryRow(query, id, product.Name, product.Price, 
		product.Quantity, metadataJSON, relatedJSON, categoriesJSON).
		Scan(&product.UpdatedAt)
}

func (r *ProductRepository) Delete(id uuid.UUID) error {
	query := `DELETE FROM products WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	
	return nil
}

func (r *ProductRepository) BulkCreate(products []models.Product) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO products (id, name, price, quantity, metadata, related_ids, categories)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	
	stmt, err := tx.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i := range products {
		products[i].ID = uuid.New()
		metadataJSON, _ := json.Marshal(products[i].Metadata)
		relatedJSON, _ := json.Marshal(products[i].RelatedIDs)
		categoriesJSON, _ := json.Marshal(products[i].Categories)

		_, err := stmt.Exec(products[i].ID, products[i].Name, products[i].Price,
			products[i].Quantity, metadataJSON, relatedJSON, categoriesJSON)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}