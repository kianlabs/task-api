package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"task-api/internal/auth"
	"task-api/internal/model"
	"task-api/internal/repository"
	"task-api/internal/service"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func setupTestDB(t *testing.T) *sql.DB {
	connStr := os.Getenv("TEST_DATABASE_URL")
	if connStr == "" {
		t.Skip("TEST_DATABASE_URL belum diset; lewati test integrasi")
	}
	parsed, err := url.Parse(connStr)
	if err != nil {
		t.Fatalf("URL database tidak valid: %v", err)
	}

	if parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "::1" {
		t.Fatalf("Database harus berada di 127.0.0.1, bukan %s", parsed.Hostname())
	}

	dbName := strings.TrimPrefix(parsed.Path, "/")
	if !strings.HasSuffix(dbName, "_test") || dbName == "taskdb" {
		t.Fatalf("Nama database harus berakhiran _test, bukan %s", dbName)
	}

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("cannot connect to test db: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("cannot ping test db: %v", err)
	}
	return db
}

const testJWTSecret = "test-secret-key"

func createTestUser(t *testing.T, db *sql.DB, email string) (int64, string) {
	var userID int64
	err := db.QueryRow("INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id", email, "dummyhash").Scan(&userID)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	token, err := auth.GenerateToken(userID, testJWTSecret)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	return userID, token
}

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	HealthHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	expectedBody := `{"status":"ok"}`
	if strings.TrimSpace(rr.Body.String()) != expectedBody {
		t.Errorf("expected body %s, got %s", expectedBody, rr.Body.String())
	}
}

func TestCreateTask(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	taskRepo := repository.NewTaskRepository(db)
	taskService := service.NewTaskService(taskRepo)
	h := NewTaskHandler(db, taskService)

	_, token := createTestUser(t, db, fmt.Sprintf("create_%d@example.com", time.Now().UnixNano()))
	authMiddleware := auth.Middleware(testJWTSecret)
	handler := authMiddleware(http.HandlerFunc(h.CreateTask))

	body := `{"title":"Belajar Unit Test","status":"pending","deadline":"2026-10-30"}`
	req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	req.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}
}

func TestCreateTask_Invalid(t *testing.T) {
	taskService := service.NewTaskService(nil)
	h := NewTaskHandler(nil, taskService)

	token, err := auth.GenerateToken(1, testJWTSecret)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	authMiddleware := auth.Middleware(testJWTSecret)
	handler := authMiddleware(http.HandlerFunc(h.CreateTask))

	body := `{"title":"","status":"pending","deadline":"2026-10-30"}`
	req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	req.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestGetTaskByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	taskRepo := repository.NewTaskRepository(db)
	taskService := service.NewTaskService(taskRepo)
	h := NewTaskHandler(db, taskService)

	_, token := createTestUser(t, db, fmt.Sprintf("create_%d@example.com", time.Now().UnixNano()))
	authMiddleware := auth.Middleware(testJWTSecret)

	mux := http.NewServeMux()
	mux.Handle("POST /tasks", authMiddleware(http.HandlerFunc(h.CreateTask)))
	mux.Handle("GET /tasks/{id}", authMiddleware(http.HandlerFunc(h.GetTaskByID)))

	createBody := `{"title":"Task 1","status":"pending","deadline":"2026-10-30"}`

	createReq := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRr := httptest.NewRecorder()
	createReq.Header.Set("Authorization", "Bearer "+token)
	mux.ServeHTTP(createRr, createReq)

	notFoundReq := httptest.NewRequest(http.MethodGet, "/tasks/9999", nil)
	notFoundRr := httptest.NewRecorder()
	notFoundReq.Header.Set("Authorization", "Bearer "+token)
	mux.ServeHTTP(notFoundRr, notFoundReq)

	if notFoundRr.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, notFoundRr.Code)
	}
}

func TestAuthMiddleware_Reject(t *testing.T) {
	authMiddleware := auth.Middleware(testJWTSecret)
	nextCalled := false
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})
	handler := authMiddleware(dummyHandler)

	reqNoToken := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	rrNoToken := httptest.NewRecorder()
	handler.ServeHTTP(rrNoToken, reqNoToken)

	if rrNoToken.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for no token, got %d", rrNoToken.Code)
	}
	if nextCalled {
		t.Errorf("expected next handler not to be called when token is missing")
	}

	reqInvalidToken := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	reqInvalidToken.Header.Set("Authorization", "Bearer invalid-token-string")
	rrInvalidToken := httptest.NewRecorder()
	handler.ServeHTTP(rrInvalidToken, reqInvalidToken)

	if rrInvalidToken.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid token, got %d", rrInvalidToken.Code)
	}
	if nextCalled {
		t.Errorf("expected next handler not to be called when token is invalid")
	}
}

func TestTaskOwnership_CrossUserAccess(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	taskRepo := repository.NewTaskRepository(db)
	taskService := service.NewTaskService(taskRepo)
	h := NewTaskHandler(db, taskService)

	authMiddleware := auth.Middleware(testJWTSecret)
	mux := http.NewServeMux()
	mux.Handle("POST /tasks", authMiddleware(http.HandlerFunc(h.CreateTask)))
	mux.Handle("GET /tasks", authMiddleware(http.HandlerFunc(h.GetTask)))
	mux.Handle("GET /tasks/{id}", authMiddleware(http.HandlerFunc(h.GetTaskByID)))

	// Buat 2 user: Siti dan Budi
	_, sitiToken := createTestUser(t, db, fmt.Sprintf("create_%d@example.com", time.Now().UnixNano()))
	_, budiToken := createTestUser(t, db, fmt.Sprintf("create_%d@example.com", time.Now().UnixNano()))

	// Siti membuat task
	body := `{"title":"Task Rahasia Siti","status":"pending","deadline":"2026-11-01"}`
	createReq := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+sitiToken)
	createRr := httptest.NewRecorder()
	mux.ServeHTTP(createRr, createReq)

	if createRr.Code != http.StatusCreated {
		t.Fatalf("expected Siti create task 201, got %d", createRr.Code)
	}

	var createdTask model.Task
	if err := json.Unmarshal(createRr.Body.Bytes(), &createdTask); err != nil {
		t.Fatalf("failed to decode created task: %v", err)
	}

	// Budi mencoba mengakses task Siti lewat GET Harus 404
	budiReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tasks/%d", createdTask.ID), nil)
	budiReq.Header.Set("Authorization", "Bearer "+budiToken)
	budiRr := httptest.NewRecorder()
	mux.ServeHTTP(budiRr, budiReq)

	if budiRr.Code != http.StatusNotFound {
		t.Errorf("expected 404 when Budi accesses Siti's task, got %d", budiRr.Code)
	}

	// Budi melihat list task-nya Harus kosong []
	budiListReq := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	budiListReq.Header.Set("Authorization", "Bearer "+budiToken)
	budiListRr := httptest.NewRecorder()
	mux.ServeHTTP(budiListRr, budiListReq)

	if budiListRr.Code != http.StatusOK {
		t.Errorf("expected 200 for Budi list tasks, got %d", budiListRr.Code)
	}

	var budiTasks []model.Task
	if err := json.Unmarshal(budiListRr.Body.Bytes(), &budiTasks); err != nil {
		t.Fatalf("failed to decode Budi tasks: %v", err)
	}

	if len(budiTasks) != 0 {
		t.Errorf("expected Budi to have 0 tasks, got %d", len(budiTasks))
	}
}
