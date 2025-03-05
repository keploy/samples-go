package repository

const (
	getAllAuthors = `SELECT * FROM authors`
	// getAuthorByID = `SELECT * FROM authors WHERE id = $1`
	getBookByID = `SELECT b.id, b.title, b.year, b.author_id, a.first_name, a.last_name
							FROM books b
							LEFT JOIN authors a on b.author_id=a.id
							WHERE b.id = $1;`
	getBooksByAuthorID = `SELECT b.id, b.title, b.year, b.author_id, a.first_name, a.last_name
							FROM authors a 
							LEFT JOIN books b on a.id=b.author_id 
							WHERE a.id = $1;`
	getAllBooks = `SELECT b.id, b.title, b.year, b.author_id, a.first_name, a.last_name
							FROM books b 
							LEFT JOIN authors a on b.author_id=a.id;`
	createBookStmt   = `INSERT INTO books(title, year, author_id) VALUES ($1, $2, $3);`
	createAuthorStmt = `INSERT INTO authors(first_name,last_name) VALUES ($1, $2);`
)
