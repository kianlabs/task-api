package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"task-api/internal/auth"
	"task-api/internal/model"
	"task-api/internal/response"
	"task-api/internal/service"
)

const maxBodyBytes = 1 << 20 // 1 MiB

type TaskHandler struct {
	service *service.TaskService
}

func NewTaskHandler(service *service.TaskService) *TaskHandler {
	return &TaskHandler{service: service}
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var task model.Task
	if err := decodeJSON(w, r, &task); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if err := h.service.CreateTask(r.Context(), userID, &task); err != nil {
		writeServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, task)
}

func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	filter, err := parseTaskFilter(r)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	filter.UserID = userID

	tasks, total, err := h.service.ListTasks(r.Context(), filter)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.Paginated(w, http.StatusOK, tasks, filter.Page, filter.Limit, total)
}

func (h *TaskHandler) GetTaskByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, ok := parseIDPath(r, "id")
	if !ok {
		response.Error(w, http.StatusBadRequest, "invalid task id")
		return
	}

	task, err := h.service.GetTask(r.Context(), id, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, task)
}

func (h *TaskHandler) UpdateTaskByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, ok := parseIDPath(r, "id")
	if !ok {
		response.Error(w, http.StatusBadRequest, "invalid task id")
		return
	}

	var updated model.Task
	if err := decodeJSON(w, r, &updated); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	task, err := h.service.UpdateTask(r.Context(), id, userID, &updated)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, task)
}

func (h *TaskHandler) DeleteTaskByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, ok := parseIDPath(r, "id")
	if !ok {
		response.Error(w, http.StatusBadRequest, "invalid task id")
		return
	}

	if err := h.service.DeleteTask(r.Context(), id, userID); err != nil {
		writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TaskHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	stats, err := h.service.Dashboard(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, stats)
}

// parseTaskFilter membaca query filter dan menolak nilai enum yang tidak dikenal.
func parseTaskFilter(r *http.Request) (model.TaskFilter, error) {
	query := r.URL.Query()

	filter := model.TaskFilter{
		Status:   query.Get("status"),
		Priority: query.Get("priority"),
		Search:   query.Get("q"),
		Sort:     query.Get("sort"),
		Order:    query.Get("order"),
		Page:     parseIntQuery(r, "page", 1, 1, 0),
		Limit:    parseIntQuery(r, "limit", 20, 1, 100),
	}

	if filter.Status != "" && !model.IsValidStatus(filter.Status) {
		return model.TaskFilter{}, errInvalidFilter("status must be one of: todo, in_progress, done")
	}
	if filter.Priority != "" && !model.IsValidPriority(filter.Priority) {
		return model.TaskFilter{}, errInvalidFilter("priority must be one of: low, medium, high, urgent")
	}
	if raw := query.Get("project_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			return model.TaskFilter{}, errInvalidFilter("project_id must be a positive integer")
		}
		filter.ProjectID = &id
	}

	filter.Normalize()
	return filter, nil
}

type filterError string

func (e filterError) Error() string { return string(e) }

func errInvalidFilter(msg string) error { return filterError(msg) }

// decodeJSON membatasi ukuran body dan menolak field yang tidak dikenal.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}
