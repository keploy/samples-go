// Package repositories contains logic to interact with database uisng raw SQL queries
package repositories

import (
	"database/sql"

	"github.com/saketV8/book-store-inventory/pkg/models"
)

type BookModel struct {
	DB *sql.DB
}

func (bookModel *BookModel) GetBooks() ([]models.Book, error) {
	statement := `SELECT book_id, title, author, price, published_date, stock_quantity FROM books ORDER BY book_id DESC`

	rows, err := bookModel.DB.Query(statement)
	if err != nil {
		return nil, err
	}

	//initializing the empty slice <Book> of data type <models.Book>
	books := []models.Book{}
	for rows.Next() {
		// initializing the empty variable <book> of data type <models.Book>
		book := models.Book{}
		//extracting data from rows and setting in <models.Book>
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

		// adding books to the books slice/array
		books = append(books, book)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	// final return
	return books, nil
}

func (bookModel *BookModel) GetBookByID(bookID string) (models.Book, error) {
	statement := `SELECT book_id, title, author, price, published_date, stock_quantity FROM books WHERE book_id = ? ;`
	// empty book model
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
		return book, err
	}

	return book, nil
}

func (bookModel *BookModel) AddBook(book models.Book) (models.Book, error) {
	statement := `INSERT INTO books (title, author, price, published_date, stock_quantity) VALUES (?, ?, ?, ?, ?)`

	result, err := bookModel.DB.Exec(statement, book.Title, book.Author, book.Price, book.PublishedDate, book.StockQuantity)
	if err != nil {
		return models.Book{}, err
	}

	lastInsertedID, err := result.LastInsertId()
	if err != nil {
		return models.Book{}, err
	}

	// adding Id to book model inserted autmatically by sqlite before returning to end user
	// removing int gives int64 cannot covert to int :)
	book.BookID = int(lastInsertedID)

	return book, nil
}

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

func (bookModel *BookModel) UpdateBook(book models.BookUpdateRequestBody) (int, error) {
	statement := `UPDATE books SET title = ?, author = ?, price = ?, published_date = ?, stock_quantity = ? WHERE book_id = ?`

	result, err := bookModel.DB.Exec(statement, book.Title, book.Author, book.Price, book.PublishedDate, book.StockQuantity, book.BookID)
	if err != nil {
		return 0, err
	}

	rowAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(rowAffected), nil
}
