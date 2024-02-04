// model.go
package main

import (
	"context"
	"database/sql"
)

type product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

func (p *product) getProduct(ctx context.Context, db *sql.DB) error {
	return db.QueryRowContext(ctx, "SELECT name, price FROM products WHERE id=$1",
		p.ID).Scan(&p.Name, &p.Price)
}

func (p *product) updateProduct(ctx context.Context, db *sql.DB) error {
	_, err :=
		db.ExecContext(ctx, "UPDATE products SET name=$1, price=$2 WHERE id=$3",
			p.Name, p.Price, p.ID)

	return err
}

func (p *product) deleteProduct(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, "DELETE FROM products WHERE id=$1", p.ID)

	return err
}

func (p *product) createProduct(ctx context.Context, db *sql.DB) error {
	err := db.QueryRowContext(ctx,
		"INSERT INTO products(name, price) VALUES($1, $2) RETURNING id",
		p.Name, p.Price).Scan(&p.ID)

	if err != nil {
		return err
	}

	return nil
}

func getProducts(ctx context.Context, db *sql.DB, start, count int) ([]product, error) {
	rows, err := db.QueryContext(ctx,
		"SELECT id, name,  price FROM products LIMIT $1 OFFSET $2",
		count, start)

	if err != nil {
		return nil, err
	}

	defer handleDeferError(rows.Close())

	products := []product{}

	for rows.Next() {
		var p product
		if err := rows.Scan(&p.ID, &p.Name, &p.Price); err != nil {
			return nil, err
		}
		products = append(products, p)
	}

	return products, nil
}
