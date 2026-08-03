package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ID         uuid.UUID              `json:"id" db:"id"`
	Name       string                 `json:"name" db:"name"`
	Price      float64                `json:"price" db:"price"`
	Quantity   int                    `json:"quantity" db:"quantity"`
	Metadata   map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	RelatedIDs []string               `json:"related_ids,omitempty" db:"related_ids"`
	Categories []string               `json:"categories,omitempty" db:"categories"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at" db:"updated_at"`
}

type ProductRating struct {
	ID        uuid.UUID `json:"id" db:"id"`
	ProductID uuid.UUID `json:"product_id" db:"product_id"`
	Score     float64   `json:"score" db:"score"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type ProductTag struct {
	ID        uuid.UUID `json:"id" db:"id"`
	ProductID uuid.UUID `json:"product_id" db:"product_id"`
	Tag       string    `json:"tag" db:"tag"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// JSONB implements the driver.Valuer and sql.Scanner interfaces for PostgreSQL JSONB
type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = make(map[string]interface{})
		return nil
	}
	return json.Unmarshal(value.([]byte), j)
}

// StringSlice implements the driver.Valuer and sql.Scanner interfaces for string slices
type StringSlice []string

func (s StringSlice) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = []string{}
		return nil
	}
	return json.Unmarshal(value.([]byte), s)
}