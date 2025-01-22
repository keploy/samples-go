// Package handlers provides HTTP request handlers for managing authors and books.
package handlers

import (
	"context"
	"encoding/json"
	"fasthttp-postgres/internal/entity"
	"net/http"
	"strconv"

	"github.com/valyala/fasthttp"
)

type Handler struct {
	repository Repository
}

type Repository interface {
	GetAllAuthors(context.Context) ([]entity.Author, error)
	GetAllBooks(context.Context) ([]entity.Book, error)
	GetBookByID(context.Context, int) ([]entity.Book, error)
	GetBooksByAuthorID(context.Context, int) ([]entity.Book, error)
	CreateBook(context.Context, entity.Book) error
	CreateAuthor(context.Context, entity.Author) error
}

func NewHandler(repository Repository) *Handler {
	return &Handler{
		repository: repository,
	}
}

func (h *Handler) GetAllAuthors(ctx *fasthttp.RequestCtx) {
	authors, err := h.repository.GetAllAuthors(ctx)
	if err != nil {
		sendError(ctx, err, 500)
		return
	}
	sendData(ctx, authors)
}

func (h *Handler) GetAllBooks(ctx *fasthttp.RequestCtx) {
	books, err := h.repository.GetAllBooks(ctx)
	if err != nil {
		sendError(ctx, err, 500)
		return
	}
	sendData(ctx, books)
}

func (h *Handler) GetBookByID(ctx *fasthttp.RequestCtx) {
	bookID := ctx.UserValue("id").(string)
	id, err := strconv.Atoi(bookID)
	if err != nil {
		sendError(ctx, nil, http.StatusNotFound)
		return
	}
	books, err := h.repository.GetBookByID(ctx, id)
	if err != nil {
		sendError(ctx, nil, http.StatusNotFound)
		return
	}
	sendData(ctx, books[0])
}

func (h *Handler) GetBooksByAuthorID(ctx *fasthttp.RequestCtx) {
	authorID := ctx.UserValue("id").(string)
	id, err := strconv.Atoi(authorID)
	if err != nil {
		sendError(ctx, nil, http.StatusNotFound)
		return
	}
	books, err := h.repository.GetBooksByAuthorID(ctx, id)
	if err != nil {
		sendError(ctx, nil, http.StatusNotFound)
		return
	}
	sendData(ctx, books)
}

func (h *Handler) CreateBook(ctx *fasthttp.RequestCtx) {
	var req createBookRequest
	err := json.Unmarshal(ctx.Request.Body(), &req)
	if err != nil {
		sendError(ctx, err, http.StatusBadRequest)
		return
	}
	if err := h.repository.CreateBook(ctx, req.convert()); err != nil {
		sendError(ctx, err, http.StatusInternalServerError)
		return
	}
	ctx.SetStatusCode(http.StatusCreated)
}

func (h *Handler) CreateAuthor(ctx *fasthttp.RequestCtx) {
	var author entity.Author
	err := json.Unmarshal(ctx.Request.Body(), &author)
	if err != nil {
		sendError(ctx, err, http.StatusBadRequest)
		return
	}
	if err := h.repository.CreateAuthor(ctx, author); err != nil {
		sendError(ctx, err, http.StatusInternalServerError)
		return
	}
	ctx.SetStatusCode(http.StatusCreated)
}

type createBookRequest struct {
	Title    string `json:"title"`
	Year     int    `json:"year"`
	AuthorID uint   `json:"author_id"`
}

func (r createBookRequest) convert() entity.Book {
	return entity.Book{
		Title: r.Title,
		Year:  r.Year,
		Author: entity.Author{
			ID: r.AuthorID,
		},
	}
}

func sendData(ctx *fasthttp.RequestCtx, data interface{}) {
	ctx.SetContentType("application/json")
	v, err := json.Marshal(data)
	if err != nil {
		sendError(ctx, err, 500)
		return
	}
	ctx.Response.SetBody(v)
}

func sendError(ctx *fasthttp.RequestCtx, err error, code int) {
	var msg string
	if err != nil {
		msg = err.Error()
	}
	ctx.Error(msg, code)
}
