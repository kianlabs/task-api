package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"task-api/internal/auth"
	"task-api/internal/model"
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
	_, _ = db.Exec("DELETE FROM users")

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
	if rr.Body.String() != expectedBody {
		t.Errorf("expected body %s, got %s", expectedBody, rr.Body.String())
	}
}

func TestCreateTask(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	h := NewTaskHandler(db)

	_, token := createTestUser(t, db, "testuser@example.com")
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
	db := setupTestDB(t)
	defer db.Close()
	h := NewTaskHandler(db)

	_, token := createTestUser(t, db, "testuser@example.com")
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
	h := NewTaskHandler(db)

	_, token := createTestUser(t, db, "gettask@example.com")
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
	db := setupTestDB(t)
	defer db.Close()
	h := NewTaskHandler(db)

	authMiddleware := auth.Middleware(testJWTSecret)
	handler := authMiddleware(http.HandlerFunc(h.GetTask))

	reqNoToken := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	rrNoToken := httptest.NewRecorder()
	handler.ServeHTTP(rrNoToken, reqNoToken)

	if rrNoToken.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for no token, got %d", rrNoToken.Code)
	}

	reqInvalidToken := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	reqInvalidToken.Header.Set("Authorization", "Bearer invalid-token-string")
	rrInvalidToken := httptest.NewRecorder()
	handler.ServeHTTP(rrInvalidToken, reqInvalidToken)

	if rrInvalidToken.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid token, got %d", rrInvalidToken.Code)
	}
}

func TestTaskOwnership_CrossUserAccess(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	h := NewTaskHandler(db)

	authMiddleware := auth.Middleware(testJWTSecret)
	mux := http.NewServeMux()
	mux.Handle("POST /tasks", authMiddleware(http.HandlerFunc(h.CreateTask)))
	mux.Handle("GET /tasks", authMiddleware(http.HandlerFunc(h.GetTask)))
	mux.Handle("GET /tasks/{id}", authMiddleware(http.HandlerFunc(h.GetTaskByID)))

	// 1. Buat 2 user: Siti dan Budi
	_, sitiToken := createTestUser(t, db, "siti@example.com")
	_, budiToken := createTestUser(t, db, "budi@example.com")

	// 2. Siti membuat task
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

	// 3. Budi mencoba mengakses task Siti lewat GET /tasks/{id} -> Harus 404
	budiReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tasks/%d", createdTask.ID), nil)
	budiReq.Header.Set("Authorization", "Bearer "+budiToken)
	budiRr := httptest.NewRecorder()
	mux.ServeHTTP(budiRr, budiReq)

	if budiRr.Code != http.StatusNotFound {
		t.Errorf("expected 404 when Budi accesses Siti's task, got %d", budiRr.Code)
	}

	// 4. Budi melihat list task-nya -> Harus kosong []
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
