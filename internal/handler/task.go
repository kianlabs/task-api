package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"task-api/internal/auth"
	"task-api/internal/model"
)

type TaskHandler struct {
	db *sql.DB
}

func NewTaskHandler(db *sql.DB) *TaskHandler {
	return &TaskHandler{db: db}
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var task model.Task
	w.Header().Set("Content-Type", "application/json")

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&task); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request payload"})
		return
	}

	if err := model.ValidateTask(task); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}
	task.UserID = userID

	query := `INSERT INTO tasks (user_id, title, status, deadline)
	          VALUES ($1, $2, $3, $4)
	          RETURNING id`

	err := h.db.QueryRowContext(r.Context(), query, task.UserID, task.Title, task.Status, task.Deadline).Scan(&task.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create task"})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}

	query := `SELECT id, user_id, title, status, deadline FROM tasks WHERE user_id = $1 ORDER BY id`
	rows, err := h.db.QueryContext(r.Context(), query, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch tasks"})
		return
	}
	defer rows.Close()

	tasks := []model.Task{}
	for rows.Next() {
		var t model.Task
		if err := rows.Scan(&t.ID, &t.UserID, &t.Title, &t.Status, &t.Deadline); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to scan task"})
			return
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to read tasks"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tasks)
}

func (h *TaskHandler) GetTaskByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid task ID"})
		return
	}

	var t model.Task
	query := `SELECT id, user_id, title, status, deadline FROM tasks WHERE id = $1 AND user_id = $2`
	err = h.db.QueryRowContext(r.Context(), query, id, userID).Scan(&t.ID, &t.UserID, &t.Title, &t.Status, &t.Deadline)

	if errors.Is(err, sql.ErrNoRows) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Task not found"})
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch task"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(t)
}

func (h *TaskHandler) UpdateTaskByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid task ID"})
		return
	}

	var updated model.Task
	if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request payload"})
		return
	}

	if err := model.ValidateTask(updated); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	query := `UPDATE tasks SET title = $1, status = $2, deadline = $3 WHERE id = $4 AND user_id = $5`
	res, err := h.db.ExecContext(r.Context(), query, updated.Title, updated.Status, updated.Deadline, id, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update task"})
		return
	}

	affected, err := res.RowsAffected()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to check update"})
		return
	}

	if affected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Task not found"})
		return
	}

	updated.ID = id
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updated)
}

func (h *TaskHandler) DeleteTaskByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid task ID"})
		return
	}

	res, err := h.db.ExecContext(r.Context(), `DELETE FROM tasks WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete task"})
		return
	}

	affected, err := res.RowsAffected()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to check delete"})
		return
	}
	if affected == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Task not found"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Task deleted successfully"})
}
