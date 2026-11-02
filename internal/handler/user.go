package handler

import (
	"net/http"

	"task-api/internal/auth"
	"task-api/internal/response"
	"task-api/internal/service"
)

type UserHandler struct {
	service *service.UserService
}

func NewUserHandler(service *service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.service.GetProfile(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, user)
}

func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var payload struct {
		Email string `json:"email"`
	}
	if err := decodeJSON(w, r, &payload); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if payload.Email == "" {
		response.Error(w, http.StatusBadRequest, "email is required")
		return
	}

	user, err := h.service.UpdateEmail(r.Context(), userID, payload.Email)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, user)
}

func (h *UserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var payload struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := decodeJSON(w, r, &payload); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if payload.CurrentPassword == "" || payload.NewPassword == "" {
		response.Error(w, http.StatusBadRequest, "current_password and new_password are required")
		return
	}

	if err := h.service.ChangePassword(r.Context(), userID, payload.CurrentPassword, payload.NewPassword); err != nil {
		writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
