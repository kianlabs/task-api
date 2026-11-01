package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"task-api/internal/auth"
	"task-api/internal/model"
	"task-api/internal/response"
	"task-api/internal/service"
)

type TaskHandler struct {
	db      *sql.DB
	service *service.TaskService
}

func NewTaskHandler(db *sql.DB, service *service.TaskService) *TaskHandler {
	return &TaskHandler{
		db:      db,
		service: service,
	}
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var task model.Task

	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if err := model.ValidateTask(task); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}

	task.UserID = userID

	if err := h.service.CreateTask(r.Context(), &task); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to create task")
		return
	}

	response.JSON(w, http.StatusCreated, task)
}

func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	tasks, err := h.service.GetTasks(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to fetch tasks")
		return
	}

	response.JSON(w, http.StatusOK, tasks)
}

func (h *TaskHandler) GetTaskByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid task ID")
		return
	}

	task, err := h.service.GetTaskByID(r.Context(), id, userID)
	if errors.Is(err, sql.ErrNoRows) {
		response.Error(w, http.StatusNotFound, "Task not found")
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to fetch task")
		return
	}

	response.JSON(w, http.StatusOK, task)
}

func (h *TaskHandler) UpdateTaskByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid task ID")
		return
	}

	var updated model.Task
	if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	found, err := h.service.UpdateTask(r.Context(), id, userID, &updated)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if !found {
		response.Error(w, http.StatusNotFound, "Task not found")
		return
	}

	updated.ID = id
	response.JSON(w, http.StatusOK, updated)
}

func (h *TaskHandler) DeleteTaskByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid task ID")
		return
	}

	found, err := h.service.DeleteTask(r.Context(), id, userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to delete task")
		return
	}
	if !found {
		response.Error(w, http.StatusNotFound, "Task not found")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Task deleted successfully"})
}
