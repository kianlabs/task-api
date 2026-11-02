package handler

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"task-api/internal/auth"
	"task-api/internal/model"
)

func TestHealthHandler(t *testing.T) {
	req := newRequest(http.MethodGet, "/health")
	rr := recordRequest(HealthHandler, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if strings.TrimSpace(rr.Body.String()) != `{"status":"ok"}` {
		t.Errorf("unexpected body: %s", rr.Body.String())
	}
}

func TestTaskCRUDAndPagination(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	router := newTestRouter(db)

	_, token := createTestUser(t, db)

	// Create 3 task
	for i := 1; i <= 3; i++ {
		rr := doRequest(t, router, requestOptions{
			method: http.MethodPost,
			path:   "/tasks",
			token:  token,
			body: map[string]any{
				"title":    fmt.Sprintf("Task %d", i),
				"status":   model.StatusTodo,
				"priority": model.PriorityMedium,
				"deadline": "2026-12-01",
			},
		})
		if rr.Code != http.StatusCreated {
			t.Fatalf("create task %d: expected 201, got %d (%s)", i, rr.Code, rr.Body.String())
		}
	}

	// List dengan pagination limit 2
	rr := doRequest(t, router, requestOptions{method: http.MethodGet, path: "/tasks?page=1&limit=2", token: token})
	if rr.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", rr.Code)
	}
	page1 := decodeBody[listEnvelope[model.Task]](t, rr)
	if len(page1.Data) != 2 {
		t.Errorf("expected 2 tasks on page 1, got %d", len(page1.Data))
	}
	if page1.Pagination.Total < 3 {
		t.Errorf("expected total >= 3, got %d", page1.Pagination.Total)
	}
	if page1.Pagination.Page != 1 || page1.Pagination.Limit != 2 {
		t.Errorf("unexpected pagination: %+v", page1.Pagination)
	}

	rr = doRequest(t, router, requestOptions{method: http.MethodGet, path: "/tasks?page=2&limit=2", token: token})
	page2 := decodeBody[listEnvelope[model.Task]](t, rr)
	if len(page2.Data) == 0 {
		t.Error("expected page 2 to contain data")
	}

	// Get by id
	id := page1.Data[0].ID
	rr = doRequest(t, router, requestOptions{method: http.MethodGet, path: fmt.Sprintf("/tasks/%d", id), token: token})
	if rr.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", rr.Code)
	}

	// Update
	rr = doRequest(t, router, requestOptions{
		method: http.MethodPut,
		path:   fmt.Sprintf("/tasks/%d", id),
		token:  token,
		body: map[string]any{
			"title":    "Task Updated",
			"status":   model.StatusInProgress,
			"priority": model.PriorityHigh,
			"deadline": "2026-12-15",
		},
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	updated := decodeBody[model.Task](t, rr)
	if updated.Title != "Task Updated" || updated.Status != model.StatusInProgress {
		t.Errorf("update not applied: %+v", updated)
	}

	// Delete
	rr = doRequest(t, router, requestOptions{method: http.MethodDelete, path: fmt.Sprintf("/tasks/%d", id), token: token})
	if rr.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d", rr.Code)
	}

	rr = doRequest(t, router, requestOptions{method: http.MethodGet, path: fmt.Sprintf("/tasks/%d", id), token: token})
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", rr.Code)
	}
}

func TestTaskOwnership_CrossUserAccess(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	router := newTestRouter(db)

	_, sitiToken := createTestUser(t, db)
	_, budiToken := createTestUser(t, db)

	rr := doRequest(t, router, requestOptions{
		method: http.MethodPost,
		path:   "/tasks",
		token:  sitiToken,
		body: map[string]any{
			"title":    "Task Rahasia Siti",
			"status":   model.StatusTodo,
			"priority": model.PriorityLow,
			"deadline": "2026-12-01",
		},
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
	created := decodeBody[model.Task](t, rr)

	// Budi tidak boleh melihat detail
	rr = doRequest(t, router, requestOptions{method: http.MethodGet, path: fmt.Sprintf("/tasks/%d", created.ID), token: budiToken})
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cross-user access, got %d", rr.Code)
	}

	// Budi tidak boleh update
	rr = doRequest(t, router, requestOptions{
		method: http.MethodPut,
		path:   fmt.Sprintf("/tasks/%d", created.ID),
		token:  budiToken,
		body: map[string]any{
			"title": "Dibajak", "status": model.StatusDone,
			"priority": model.PriorityLow, "deadline": "2026-12-01",
		},
	})
	if rr.Code != http.StatusForbidden && rr.Code != http.StatusNotFound {
		t.Errorf("expected 403/404 for cross-user update, got %d", rr.Code)
	}

	// List Budi kosong
	rr = doRequest(t, router, requestOptions{method: http.MethodGet, path: "/tasks", token: budiToken})
	list := decodeBody[listEnvelope[model.Task]](t, rr)
	if len(list.Data) != 0 {
		t.Errorf("expected empty list for Budi, got %d", len(list.Data))
	}
}

func TestTaskFilteringAndSearch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	router := newTestRouter(db)
	_, token := createTestUser(t, db)

	tasks := []map[string]any{
		{"title": "Belajar Go", "status": model.StatusTodo, "priority": model.PriorityHigh, "deadline": "2026-12-01"},
		{"title": "Belajar SQL", "status": model.StatusDone, "priority": model.PriorityLow, "deadline": "2026-12-02"},
		{"title": "Review PR", "status": model.StatusTodo, "priority": model.PriorityUrgent, "deadline": "2026-12-03"},
	}
	for _, body := range tasks {
		rr := doRequest(t, router, requestOptions{method: http.MethodPost, path: "/tasks", token: token, body: body})
		if rr.Code != http.StatusCreated {
			t.Fatalf("seed task: %d %s", rr.Code, rr.Body.String())
		}
	}

	cases := []struct {
		query string
		want  int
	}{
		{"/tasks?status=todo", 2},
		{"/tasks?priority=urgent", 1},
		{"/tasks?q=Belajar", 2},
		{"/tasks?status=todo&priority=high", 1},
		{"/tasks?sort=priority&order=desc", 3},
	}

	for _, tc := range cases {
		rr := doRequest(t, router, requestOptions{method: http.MethodGet, path: tc.query, token: token})
		if rr.Code != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d", tc.query, rr.Code)
		}
		got := decodeBody[listEnvelope[model.Task]](t, rr)
		if len(got.Data) != tc.want {
			t.Errorf("%s: expected %d results, got %d", tc.query, tc.want, len(got.Data))
		}
	}

	// Sort priority desc: urgent harus pertama.
	rr := doRequest(t, router, requestOptions{method: http.MethodGet, path: "/tasks?sort=priority&order=desc", token: token})
	sorted := decodeBody[listEnvelope[model.Task]](t, rr)
	if len(sorted.Data) > 0 && sorted.Data[0].Priority != model.PriorityUrgent {
		t.Errorf("expected urgent first, got %s", sorted.Data[0].Priority)
	}
}

func TestTaskValidation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	router := newTestRouter(db)
	_, token := createTestUser(t, db)

	cases := []struct {
		name string
		body map[string]any
	}{
		{"empty title", map[string]any{"title": "", "status": "todo", "priority": "low", "deadline": "2026-12-01"}},
		{"invalid status", map[string]any{"title": "X", "status": "pending", "priority": "low", "deadline": "2026-12-01"}},
		{"invalid priority", map[string]any{"title": "X", "status": "todo", "priority": "super", "deadline": "2026-12-01"}},
		{"bad deadline", map[string]any{"title": "X", "status": "todo", "priority": "low", "deadline": "01-12-2026"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := doRequest(t, router, requestOptions{method: http.MethodPost, path: "/tasks", token: token, body: tc.body})
			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d (%s)", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestTaskInvalidFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	router := newTestRouter(db)
	_, token := createTestUser(t, db)

	rr := doRequest(t, router, requestOptions{method: http.MethodGet, path: "/tasks?status=ngawur", token: token})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid status filter, got %d", rr.Code)
	}
}

func TestDashboard(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	router := newTestRouter(db)
	_, token := createTestUser(t, db)

	seeds := []map[string]any{
		{"title": "A", "status": model.StatusTodo, "priority": model.PriorityHigh, "deadline": "2026-12-01"},
		{"title": "B", "status": model.StatusDone, "priority": model.PriorityLow, "deadline": "2026-12-01"},
		{"title": "C", "status": model.StatusInProgress, "priority": model.PriorityUrgent, "deadline": "2026-12-01"},
	}
	for _, body := range seeds {
		doRequest(t, router, requestOptions{method: http.MethodPost, path: "/tasks", token: token, body: body})
	}

	rr := doRequest(t, router, requestOptions{method: http.MethodGet, path: "/dashboard", token: token})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	stats := decodeBody[model.DashboardStats](t, rr)
	if stats.TotalTasks != 3 {
		t.Errorf("expected 3 total tasks, got %d", stats.TotalTasks)
	}
	if stats.ByStatus[model.StatusTodo] != 1 || stats.ByStatus[model.StatusDone] != 1 {
		t.Errorf("unexpected by_status: %+v", stats.ByStatus)
	}
	if stats.ByPriority[model.PriorityUrgent] != 1 {
		t.Errorf("unexpected by_priority: %+v", stats.ByPriority)
	}
}

func TestAuthMiddlewareReject(t *testing.T) {
	mw := auth.Middleware(testJWTSecret)
	called := false
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	rr := recordRequest(h.ServeHTTP, newRequest(http.MethodGet, "/tasks"))
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", rr.Code)
	}
	if called {
		t.Error("next handler must not be called")
	}

	req := newRequest(http.MethodGet, "/tasks")
	req.Header.Set("Authorization", "Bearer invalid-token")
	rr = recordRequest(h.ServeHTTP, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with invalid token, got %d", rr.Code)
	}
	if called {
		t.Error("next handler must not be called")
	}
}
