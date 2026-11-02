package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"task-api/internal/auth"
	"task-api/internal/repository"
	"task-api/internal/service"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const testJWTSecret = "test-secret-key-at-least-32-characters"

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

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

// Router yang sama dengan main.go, supaya test mencakup middleware dan routing.
func newTestRouter(db *sql.DB) http.Handler {
	userRepo := repository.NewUserRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	projectRepo := repository.NewProjectRepository(db)

	userService := service.NewUserService(userRepo)
	taskService := service.NewTaskService(taskRepo, projectRepo, userRepo)
	projectService := service.NewProjectService(projectRepo, userRepo)

	authHandler := NewAuthHandler(db, testJWTSecret)
	userHandler := NewUserHandler(userService)
	taskHandler := NewTaskHandler(taskService)
	projectHandler := NewProjectHandler(projectService, taskService)

	authMiddleware := auth.Middleware(testJWTSecret)
	protected := func(h http.HandlerFunc) http.Handler { return authMiddleware(h) }

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", HealthHandler)
	mux.HandleFunc("POST /register", authHandler.Register)
	mux.HandleFunc("POST /login", authHandler.Login)

	mux.Handle("GET /me", protected(userHandler.GetProfile))
	mux.Handle("PATCH /me", protected(userHandler.UpdateProfile))
	mux.Handle("PATCH /me/password", protected(userHandler.ChangePassword))

	mux.Handle("POST /tasks", protected(taskHandler.CreateTask))
	mux.Handle("GET /tasks", protected(taskHandler.ListTasks))
	mux.Handle("GET /tasks/{id}", protected(taskHandler.GetTaskByID))
	mux.Handle("PUT /tasks/{id}", protected(taskHandler.UpdateTaskByID))
	mux.Handle("DELETE /tasks/{id}", protected(taskHandler.DeleteTaskByID))

	mux.Handle("POST /projects", protected(projectHandler.CreateProject))
	mux.Handle("GET /projects", protected(projectHandler.ListProjects))
	mux.Handle("GET /projects/{id}", protected(projectHandler.GetProjectByID))
	mux.Handle("PUT /projects/{id}", protected(projectHandler.UpdateProjectByID))
	mux.Handle("DELETE /projects/{id}", protected(projectHandler.DeleteProjectByID))
	mux.Handle("GET /projects/{id}/members", protected(projectHandler.ListMembers))
	mux.Handle("POST /projects/{id}/members", protected(projectHandler.AddMember))
	mux.Handle("PATCH /projects/{id}/members/{userID}", protected(projectHandler.UpdateMemberRole))
	mux.Handle("DELETE /projects/{id}/members", protected(projectHandler.RemoveMember))
	mux.Handle("GET /projects/{id}/tasks", protected(projectHandler.ListProjectTasks))
	mux.Handle("POST /projects/{id}/tasks", protected(projectHandler.CreateProjectTask))

	mux.Handle("GET /dashboard", protected(taskHandler.Dashboard))

	return mux
}

func createTestUser(t *testing.T, db *sql.DB) (int64, string) {
	t.Helper()

	email := fmt.Sprintf("user_%d@example.com", time.Now().UnixNano())
	var userID int64
	err := db.QueryRow(
		"INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id",
		email, "$2a$10$abcdefghijklmnopqrstuv",
	).Scan(&userID)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	token, err := auth.GenerateToken(userID, testJWTSecret)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	return userID, token
}

type requestOptions struct {
	method string
	path   string
	token  string
	body   any
}

func doRequest(t *testing.T, router http.Handler, opt requestOptions) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	if opt.body != nil {
		raw, err := json.Marshal(opt.body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	}

	req := httptest.NewRequest(opt.method, opt.path, reader)
	if opt.body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if opt.token != "" {
		req.Header.Set("Authorization", "Bearer "+opt.token)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func newRequest(method, path string) *http.Request {
	return httptest.NewRequest(method, path, nil)
}

func recordRequest(h http.HandlerFunc, req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	h(rr, req)
	return rr
}

func decodeBody[T any](t *testing.T, rr *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response %q: %v", rr.Body.String(), err)
	}
	return out
}

type listEnvelope[T any] struct {
	Data       []T `json:"data"`
	Pagination struct {
		Page       int `json:"page"`
		Limit      int `json:"limit"`
		Total      int `json:"total"`
		TotalPages int `json:"total_pages"`
	} `json:"pagination"`
}
