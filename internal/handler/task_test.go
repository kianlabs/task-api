package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	HealthHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	expectedBody := `{"status":"ok"}`
	if rr.Body.String() != expectedBody {
		t.Errorf("expected body %s, got %s", expectedBody, rr.Body.String())
	}
}

func TestCreateTask(t *testing.T) {
	h := NewTaskHandler()

	body := `{"title":"Belajar Unit Test","status":"pending","deadline":"2026-10-30"}`
	req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.CreateTask(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}
}

func TestCreateTask_Invalid(t *testing.T) {
	h := NewTaskHandler()

	body := `{"title":"","status":"pending","deadline":"2026-10-30"}`
	req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.CreateTask(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestGetTaskByID(t *testing.T) {
	h := NewTaskHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /tasks", h.CreateTask)
	mux.HandleFunc("GET /tasks/{id}", h.GetTaskByID)

	createBody := `{"title":"Task 1","status":"pending","deadline":"2026-10-30"}`
	createReq := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRr := httptest.NewRecorder()
	mux.ServeHTTP(createRr, createReq)

	getReq := httptest.NewRequest(http.MethodGet, "/tasks/1", nil)
	getRr := httptest.NewRecorder()
	mux.ServeHTTP(getRr, getReq)

	if getRr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, getRr.Code)
	}

	notFoundReq := httptest.NewRequest(http.MethodGet, "/tasks/99", nil)
	notFoundRr := httptest.NewRecorder()
	mux.ServeHTTP(notFoundRr, notFoundReq)

	if notFoundRr.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, notFoundRr.Code)
	}
}
