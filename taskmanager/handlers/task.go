package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"taskmanager/ent"
	"taskmanager/ent/task"

	"github.com/gorilla/mux"
)

type TaskHandler struct {
	client *ent.Client
}

type CreateInput struct {
	Title       string
	Description *string
	Status      string
	Priority    string
	UserID      int
}

type UpdateInput struct {
	Title       *string
	Description *string
	Status      *string
	Priority    *string
}

func NewTaskHandler(client *ent.Client) *TaskHandler {
	return &TaskHandler{client: client}
}

func (h *TaskHandler) GetAllTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.client.Task.Query().All(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(tasks)
}

func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	task, err := h.client.Task.Get(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(task)
}

func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var input CreateInput
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userID := 1 // Replace this with the appropriate user ID

	creator := h.client.Task.Create().
		SetTitle(input.Title).
		SetStatus(task.Status(input.Status)).
		SetPriority(task.Priority(input.Priority)).
		SetUserID(userID)

	if input.Description != nil {
		creator.SetDescription(*input.Description)
	}

	task, err := creator.Save(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(task)
}

func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var input UpdateInput
	err = json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	updater := h.client.Task.UpdateOneID(id)

	if input.Title != nil {
		updater.SetTitle(*input.Title)
	}

	if input.Description != nil {
		updater.SetDescription(*input.Description)
	}

	if input.Status != nil {
		updater.SetStatus(task.Status(*input.Status))
	}

	if input.Priority != nil {
		updater.SetPriority(task.Priority(*input.Priority))
	}

	task, err := updater.Save(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(task)
}
func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.client.Task.DeleteOneID(id).Exec(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *TaskHandler) TaskRouter() *mux.Router {
	router := mux.NewRouter()

	router.HandleFunc("/tasks", h.GetAllTasks).Methods("GET")
	router.HandleFunc("/tasks/{id:[0-9]+}", h.GetTask).Methods("GET")
	router.HandleFunc("/tasks", h.CreateTask).Methods("POST")
	router.HandleFunc("/tasks/{id:[0-9]+}", h.UpdateTask).Methods("PUT")
	router.HandleFunc("/tasks/{id:[0-9]+}", h.DeleteTask).Methods("DELETE")

	return router
}
