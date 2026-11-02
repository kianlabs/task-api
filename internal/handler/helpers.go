package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"task-api/internal/response"
	"task-api/internal/service"
)

func writeServiceError(w http.ResponseWriter, err error) {
	var appErr *service.AppError
	if errors.As(err, &appErr) {
		switch {
		case errors.Is(appErr.Kind, service.ErrNotFound):
			response.Error(w, http.StatusNotFound, appErr.Message)
		case errors.Is(appErr.Kind, service.ErrForbidden):
			response.Error(w, http.StatusForbidden, appErr.Message)
		case errors.Is(appErr.Kind, service.ErrConflict):
			response.Error(w, http.StatusConflict, appErr.Message)
		case errors.Is(appErr.Kind, service.ErrValidation):
			response.Error(w, http.StatusBadRequest, appErr.Message)
		default:
			response.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	slog.Error("unhandled error", "error", err)
	response.Error(w, http.StatusInternalServerError, "internal server error")
}

func parseIntQuery(r *http.Request, key string, fallback, min, max int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if value < min {
		return min
	}
	if max > 0 && value > max {
		return max
	}
	return value
}

func parseIDPath(r *http.Request, key string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(key), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}
