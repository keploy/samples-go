package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
)

type Handler struct {
	db *sql.DB
}

type Company struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

type CreateCompanyRequest struct {
	Name string `json:"name"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func New(db *sql.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) CreateCompany(w http.ResponseWriter, r *http.Request) {
	var req CreateCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "name is required"})
		return
	}

	var company Company
	err := h.db.QueryRow(
		"INSERT INTO companies (name) VALUES ($1) RETURNING id, name, created_at",
		req.Name,
	).Scan(&company.ID, &company.Name, &company.CreatedAt)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			writeJSON(w, http.StatusConflict, ErrorResponse{Error: "company already exists"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to create company"})
		return
	}

	writeJSON(w, http.StatusCreated, company)
}

func (h *Handler) ListCompanies(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query("SELECT id, name, created_at FROM companies ORDER BY id")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to list companies"})
		return
	}
	defer rows.Close()

	companies := []Company{}
	for rows.Next() {
		var c Company
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to scan company"})
			return
		}
		companies = append(companies, c)
	}

	writeJSON(w, http.StatusOK, companies)
}

func (h *Handler) GetCompany(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/companies/")
	if name == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "company name is required"})
		return
	}

	var c Company
	err := h.db.QueryRow("SELECT id, name, created_at FROM companies WHERE name = $1", name).
		Scan(&c.ID, &c.Name, &c.CreatedAt)
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "company not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to get company"})
		return
	}

	writeJSON(w, http.StatusOK, c)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
