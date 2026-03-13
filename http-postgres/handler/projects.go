package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// pgQuote escapes a string value for direct embedding in SQL.
// This mimics the simple query protocol used by Python/SQLAlchemy which
// embeds values as string literals rather than using $1 parameters.
func pgQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

type Project struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type CreateProjectRequest struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type UpdateProjectRequest struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "name is required"})
		return
	}
	if req.Status == "" {
		req.Status = "active"
	}

	projectID := uuid.New().String()

	var project Project
	query := fmt.Sprintf(
		"INSERT INTO projects (id, name, status) VALUES (%s::UUID, %s, %s) RETURNING id, name, status, created_at, updated_at",
		pgQuote(projectID), pgQuote(req.Name), pgQuote(req.Status),
	)
	err := h.db.QueryRow(query).Scan(&project.ID, &project.Name, &project.Status, &project.CreatedAt, &project.UpdatedAt)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			writeJSON(w, http.StatusConflict, ErrorResponse{Error: "project already exists"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to create project"})
		return
	}

	w.Header().Set("X-Project-Id", project.ID)
	writeJSON(w, http.StatusCreated, project)
}

func (h *Handler) GetProject(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/projects/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "project id is required"})
		return
	}

	var p Project
	query := fmt.Sprintf(
		"SELECT id, name, status, created_at, updated_at FROM projects WHERE id = %s::UUID",
		pgQuote(id),
	)
	err := h.db.QueryRow(query).Scan(&p.ID, &p.Name, &p.Status, &p.CreatedAt, &p.UpdatedAt)

	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "project not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to get project"})
		return
	}

	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/projects/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "project id is required"})
		return
	}

	var req UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	var p Project
	query := fmt.Sprintf(
		"UPDATE projects SET name = %s, status = %s, updated_at = NOW() WHERE id = %s::UUID RETURNING id, name, status, created_at, updated_at",
		pgQuote(req.Name), pgQuote(req.Status), pgQuote(id),
	)
	err := h.db.QueryRow(query).Scan(&p.ID, &p.Name, &p.Status, &p.CreatedAt, &p.UpdatedAt)

	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "project not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to update project"})
		return
	}

	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) ListProjects(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query("SELECT id, name, status, created_at, updated_at FROM projects ORDER BY created_at")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to list projects"})
		return
	}
	defer rows.Close()

	projects := []Project{}
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to scan project"})
			return
		}
		projects = append(projects, p)
	}

	writeJSON(w, http.StatusOK, projects)
}
