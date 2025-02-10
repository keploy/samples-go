-- +goose Up
-- +goose StatementBegin
-- Books Table
CREATE TABLE books (
    book_id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL UNIQUE,
    author TEXT NOT NULL,
    price REAL NOT NULL,
    published_date DATE,
    stock_quantity INTEGER DEFAULT 0
);

INSERT INTO books (title, author, price, published_date, stock_quantity) 
VALUES ('The Catcher in the Rye', 'J.D. Salinger', 749.99, '1951-07-16', 10);

INSERT INTO books (title, author, price, published_date, stock_quantity)
VALUES ('To Kill a Mockingbird', 'Harper Lee', 929.99, '1960-07-11', 15);

INSERT INTO books (title, author, price, published_date, stock_quantity) 
VALUES ('1984', 'George Orwell', 89.99, '1949-06-08', 8);

INSERT INTO books (title, author, price, published_date, stock_quantity) 
VALUES ('The Great Gatsby', 'F. Scott Fitzgerald', 799.99, '1925-04-10', 5);

INSERT INTO books (title, author, price, published_date, stock_quantity) 
VALUES ('Moby-Dick', 'Herman Melville', 899.99, '1851-10-18', 12);


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE books;
-- +goose StatementEnd
