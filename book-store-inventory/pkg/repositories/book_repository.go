// Package repositories contains logic to interact with database using raw SQL queries
package repositories

import (
	"database/sql"
	"errors"

	"github.com/saketV8/book-store-inventory/pkg/models"
)

type BookModel struct {
	DB *sql.DB
}

// GetBooks fetches all books from the database.
func (bookModel *BookModel) GetBooks() ([]models.Book, error) {
	statement := `SELECT book_id, title, author, price, published_date, stock_quantity FROM books ORDER BY book_id DESC`

	rows, err := bookModel.DB.Query(statement)
	if err != nil {
		return nil, err
	}
	defer rows.Close() // ✅ Prevent resource leak

	books := []models.Book{}
	for rows.Next() {
		book := models.Book{}
		err := rows.Scan(
			&book.BookID,
			&book.Title,
			&book.Author,
			&book.Price,
			&book.PublishedDate,
			&book.StockQuantity,
		)
		if err != nil {
			return nil, err
		}
		books = append(books, book)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return books, nil
}

// GetBookByID fetches a book by its ID.
func (bookModel *BookModel) GetBookByID(bookID string) (models.Book, error) {
	if bookID == "" {
		return models.Book{}, errors.New("empty book ID")
	}

	statement := `SELECT book_id, title, author, price, published_date, stock_quantity FROM books WHERE book_id = ?`

	book := models.Book{}
	err := bookModel.DB.QueryRow(statement, bookID).Scan(
		&book.BookID,
		&book.Title,
		&book.Author,
		&book.Price,
		&book.PublishedDate,
		&book.StockQuantity,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Pass this explicitly so the handler can handle 404 logic
			return models.Book{}, sql.ErrNoRows
		}
		return models.Book{}, err
	}

	return book, nil
}

// AddBook inserts a new book into the database.
func (bookModel *BookModel) AddBook(book models.Book) (models.Book, error) {
	statement := `INSERT INTO books (title, author, price, published_date, stock_quantity) VALUES (?, ?, ?, ?, ?)`

	result, err := bookModel.DB.Exec(statement,
		book.Title,
		book.Author,
		book.Price,
		book.PublishedDate,
		book.StockQuantity,
	)
	if err != nil {
		return models.Book{}, err
	}

	lastInsertedID, err := result.LastInsertId()
	if err != nil {
		return models.Book{}, err
	}

	book.BookID = int(lastInsertedID)
	return book, nil
}

// DeleteBook removes a book from the database.
func (bookModel *BookModel) DeleteBook(book models.BookDeleteRequestBody) (int, error) {
	statement := `DELETE FROM books WHERE book_id = ?;`

	result, err := bookModel.DB.Exec(statement, book.BookID)
	if err != nil {
		return 0, err
	}

	rowAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(rowAffected), nil
}

// UpdateBook updates book details in the database.
func (bookModel *BookModel) UpdateBook(book models.BookUpdateRequestBody) (int, error) {
	statement := `UPDATE books SET title = ?, author = ?, price = ?, published_date = ?, stock_quantity = ? WHERE book_id = ?`

	result, err := bookModel.DB.Exec(statement,
		book.Title,
		book.Author,
		book.Price,
		book.PublishedDate,
		book.StockQuantity,
		book.BookID,
	)
	if err != nil {
		return 0, err
	}

	rowAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(rowAffected), nil
}
