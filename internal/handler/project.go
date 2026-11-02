package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"task-api/internal/auth"
	"task-api/internal/model"
	"task-api/internal/response"
	"task-api/internal/service"
)

type ProjectHandler struct {
	service *service.ProjectService
}

func NewProjectHandler(service *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{service: service}
}

func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	var p model.Project
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := h.service.CreateProject(r.Context(), &p, userID); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	response.JSON(w, http.StatusCreated, p)
}

func (h *ProjectHandler) GetProjects(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	projects, err := h.service.GetProjects(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to fetch projects")
		return
	}
	response.JSON(w, http.StatusOK, projects)
}

func (h *ProjectHandler) GetProjectByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid project ID")
		return
	}
	project, err := h.service.GetProjectByID(r.Context(), id, userID)
	if errors.Is(err, sql.ErrNoRows) {
		response.Error(w, http.StatusNotFound, "Project not found")
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to fetch project")
		return
	}
	response.JSON(w, http.StatusOK, project)
}

func (h *ProjectHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	currentUserID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	projectID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid project ID")
		return
	}

	var member model.ProjectMember
	if err := json.NewDecoder(r.Body).Decode(&member); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if err := h.service.AddMember(r.Context(), projectID, currentUserID, &member); err != nil {
		if strings.Contains(err.Error(), "forbidden") {
			response.Error(w, http.StatusForbidden, err.Error())
			return
		}
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	response.JSON(w, http.StatusCreated, member)
}
