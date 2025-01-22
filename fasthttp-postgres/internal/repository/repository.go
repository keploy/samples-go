package repository

import (
	"context"
	"database/sql"
	"fasthttp-postgres/internal/entity"
	"fmt"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) GetAllAuthors(ctx context.Context) ([]entity.Author, error) {
	var mm []entity.Author
	rows, err := r.db.QueryContext(ctx, getAllAuthors)
	if err != nil {
		return nil, err
	}
	var m entity.Author
	for rows.Next() {
		if err = rows.Scan(&m.ID, &m.FirstName, &m.LastName); err != nil {
			return nil, err
		}
		mm = append(mm, m)
	}
	return mm, nil
}

func (r *Repository) GetAllBooks(ctx context.Context) ([]entity.Book, error) {
	var mm models
	rows, err := r.db.QueryContext(ctx, getAllBooks)
	if err != nil {
		return nil, err
	}
	var m model
	for rows.Next() {
		if err = rows.Scan(&m.ID, &m.Title, &m.Year, &m.AuthorID, &m.FirstName, &m.LastName); err != nil {
			return nil, err
		}
		mm = append(mm, m)
	}
	return mm.convert(), nil
}

func (r *Repository) GetBookByID(ctx context.Context, id int) ([]entity.Book, error) {
	var mm models
	rows, err := r.db.QueryContext(ctx, getBookByID, id)
	if err != nil {
		fmt.Println("1")
		return nil, err
	}
	var m model
	for rows.Next() {
		if err = rows.Scan(&m.ID, &m.Title, &m.Year, &m.AuthorID, &m.FirstName, &m.LastName); err != nil {
			fmt.Println("3")
			return nil, err
		}
		mm = append(mm, m)
	}
	return mm.convert(), nil
}

func (r *Repository) GetBooksByAuthorID(ctx context.Context, id int) ([]entity.Book, error) {
	var mm models
	rows, err := r.db.QueryContext(ctx, getBooksByAuthorID, id)
	if err != nil {
		return nil, err
	}
	var m model
	for rows.Next() {
		if err = rows.Scan(&m.ID, &m.Title, &m.Year, &m.AuthorID, &m.FirstName, &m.LastName); err != nil {
			return nil, err
		}
		mm = append(mm, m)
	}
	return mm.convert(), nil
}

func (r *Repository) CreateBook(ctx context.Context, b entity.Book) error {
	m := convert(b)
	_, err := r.db.ExecContext(ctx, createBookStmt, m.Title, m.Year, m.AuthorID)
	return err
}

func (r *Repository) CreateAuthor(ctx context.Context, a entity.Author) error {
	_, err := r.db.ExecContext(ctx, createAuthorStmt, a.FirstName, a.LastName)
	return err
}
