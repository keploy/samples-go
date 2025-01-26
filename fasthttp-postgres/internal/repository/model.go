// Package repository provides functions for converting between entity models and database models.
package repository

import "fasthttp-postgres/internal/entity"

type model struct {
	ID        uint   `db:"id"`
	Title     string `db:"title"`
	Year      int    `db:"year"`
	AuthorID  uint   `db:"author_id"`
	FirstName string `db:"first_name"`
	LastName  string `db:"last_name"`
}

func convert(b entity.Book) model {
	return model{
		Title:    b.Title,
		Year:     b.Year,
		AuthorID: b.Author.ID,
	}
}

func (m model) convert() entity.Book {
	return entity.Book{
		ID:    m.ID,
		Title: m.Title,
		Year:  m.Year,
		Author: entity.Author{
			ID:        m.AuthorID,
			FirstName: m.FirstName,
			LastName:  m.LastName,
		},
	}
}

type models []model

func (mm models) convert() []entity.Book {
	out := make([]entity.Book, 0, len(mm))
	for _, m := range mm {
		out = append(out, m.convert())
	}
	return out
}
