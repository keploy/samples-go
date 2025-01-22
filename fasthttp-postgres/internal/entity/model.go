// Package entity defines the data models for the application, including authors and books.
package entity

type Author struct {
	ID        uint   `json:"id" db:"id"`
	FirstName string `json:"first_name" db:"first_name"`
	LastName  string `json:"last_name" db:"last_name"`
}

type Book struct {
	ID     uint   `json:"id"`
	Title  string `json:"title"`
	Year   int    `json:"year"`
	Author Author `json:"author"`
}
