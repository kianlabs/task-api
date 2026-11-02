package handler

import (
	"net/http"

	"task-api/internal/auth"
	"task-api/internal/model"
	"task-api/internal/response"
	"task-api/internal/service"
)

type ProjectHandler struct {
	service     *service.ProjectService
	taskService *service.TaskService
}

func NewProjectHandler(service *service.ProjectService, taskService *service.TaskService) *ProjectHandler {
	return &ProjectHandler{service: service, taskService: taskService}
}

func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var project model.Project
	if err := decodeJSON(w, r, &project); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if err := h.service.CreateProject(r.Context(), userID, &project); err != nil {
		writeServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, project)
}

func (h *ProjectHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	filter := model.ProjectFilter{
		UserID: userID,
		Page:   parseIntQuery(r, "page", 1, 1, 0),
		Limit:  parseIntQuery(r, "limit", 20, 1, 100),
	}
	filter.Normalize()

	projects, total, err := h.service.ListProjects(r.Context(), filter)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.Paginated(w, http.StatusOK, projects, filter.Page, filter.Limit, total)
}

func (h *ProjectHandler) GetProjectByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, ok := parseIDPath(r, "id")
	if !ok {
		response.Error(w, http.StatusBadRequest, "invalid project id")
		return
	}

	project, err := h.service.GetProject(r.Context(), id, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, project)
}

func (h *ProjectHandler) UpdateProjectByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, ok := parseIDPath(r, "id")
	if !ok {
		response.Error(w, http.StatusBadRequest, "invalid project id")
		return
	}

	var updated model.Project
	if err := decodeJSON(w, r, &updated); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	project, err := h.service.UpdateProject(r.Context(), id, userID, &updated)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, project)
}

func (h *ProjectHandler) DeleteProjectByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, ok := parseIDPath(r, "id")
	if !ok {
		response.Error(w, http.StatusBadRequest, "invalid project id")
		return
	}

	if err := h.service.DeleteProject(r.Context(), id, userID); err != nil {
		writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ProjectHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	projectID, ok := parseIDPath(r, "id")
	if !ok {
		response.Error(w, http.StatusBadRequest, "invalid project id")
		return
	}

	members, err := h.service.ListMembers(r.Context(), projectID, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"data": members})
}

func (h *ProjectHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	projectID, ok := parseIDPath(r, "id")
	if !ok {
		response.Error(w, http.StatusBadRequest, "invalid project id")
		return
	}

	var member model.ProjectMember
	if err := decodeJSON(w, r, &member); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if member.UserID <= 0 {
		response.Error(w, http.StatusBadRequest, "user_id is required")
		return
	}

	if err := h.service.AddMember(r.Context(), projectID, userID, &member); err != nil {
		writeServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, member)
}

func (h *ProjectHandler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	projectID, ok := parseIDPath(r, "id")
	if !ok {
		response.Error(w, http.StatusBadRequest, "invalid project id")
		return
	}
	targetID, ok := parseIDPath(r, "userID")
	if !ok {
		response.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var payload struct {
		Role string `json:"role"`
	}
	if err := decodeJSON(w, r, &payload); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if err := h.service.UpdateMemberRole(r.Context(), projectID, userID, targetID, payload.Role); err != nil {
		writeServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"user_id": targetID, "role": payload.Role})
}

func (h *ProjectHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	projectID, ok := parseIDPath(r, "id")
	if !ok {
		response.Error(w, http.StatusBadRequest, "invalid project id")
		return
	}

	var payload struct {
		UserID int64 `json:"user_id"`
	}
	if err := decodeJSON(w, r, &payload); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if payload.UserID <= 0 {
		response.Error(w, http.StatusBadRequest, "user_id is required")
		return
	}

	if err := h.service.RemoveMember(r.Context(), projectID, userID, payload.UserID); err != nil {
		writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ProjectHandler) ListProjectTasks(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	projectID, ok := parseIDPath(r, "id")
	if !ok {
		response.Error(w, http.StatusBadRequest, "invalid project id")
		return
	}

	filter, err := parseTaskFilter(r)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	tasks, total, err := h.taskService.ListProjectTasks(r.Context(), projectID, userID, filter)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response.Paginated(w, http.StatusOK, tasks, filter.Page, filter.Limit, total)
}

func (h *ProjectHandler) CreateProjectTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	projectID, ok := parseIDPath(r, "id")
	if !ok {
		response.Error(w, http.StatusBadRequest, "invalid project id")
		return
	}

	var task model.Task
	if err := decodeJSON(w, r, &task); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	task.ProjectID = &projectID

	if err := h.taskService.CreateTask(r.Context(), userID, &task); err != nil {
		writeServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, task)
}
