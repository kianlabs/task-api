package handler

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func setupTestDB(t *testing.T) *sql.DB {
	connStr := "postgres://taskapi:devpassword@127.0.0.1:5432/taskdb?sslmode=disable"
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("cannot connect to test db: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("cannot ping test db: %v", err)
	}

	// Bersihkan tabel sebelum test mulai agar data terisolasi
	_, _ = db.Exec("DELETE FROM tasks")

	return db
}

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
	db := setupTestDB(t)
	defer db.Close()
	h := NewTaskHandler(db)

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
	db := setupTestDB(t)
	defer db.Close()
	h := NewTaskHandler(db)

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
	db := setupTestDB(t)
	defer db.Close()
	h := NewTaskHandler(db)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /tasks", h.CreateTask)
	mux.HandleFunc("GET /tasks/{id}", h.GetTaskByID)

	createBody := `{"title":"Task 1","status":"pending","deadline":"2026-10-30"}`
	createReq := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRr := httptest.NewRecorder()
	mux.ServeHTTP(createRr, createReq)

	notFoundReq := httptest.NewRequest(http.MethodGet, "/tasks/9999", nil)
	notFoundRr := httptest.NewRecorder()
	mux.ServeHTTP(notFoundRr, notFoundReq)

	if notFoundRr.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, notFoundRr.Code)
	}
}
