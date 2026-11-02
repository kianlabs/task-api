package handler

import (
	"fmt"
	"net/http"
	"testing"

	"task-api/internal/model"
)

func TestProjectCRUDAndPagination(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	router := newTestRouter(db)

	aliceID, aliceToken := createTestUser(t, db)

	rr := doRequest(t, router, requestOptions{
		method: http.MethodPost, path: "/projects", token: aliceToken,
		body: map[string]any{"name": "Proyek Alpha", "description": "deskripsi"},
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create project: expected 201, got %d (%s)", rr.Code, rr.Body.String())
	}
	project := decodeBody[model.Project](t, rr)
	if project.OwnerID != aliceID {
		t.Errorf("expected owner_id %d, got %d", aliceID, project.OwnerID)
	}

	rr = doRequest(t, router, requestOptions{method: http.MethodGet, path: "/projects", token: aliceToken})
	if rr.Code != http.StatusOK {
		t.Fatalf("list projects: expected 200, got %d", rr.Code)
	}
	list := decodeBody[listEnvelope[model.Project]](t, rr)
	if len(list.Data) != 1 || list.Pagination.Total != 1 {
		t.Errorf("unexpected list: %+v", list)
	}

	rr = doRequest(t, router, requestOptions{
		method: http.MethodPut, path: fmt.Sprintf("/projects/%d", project.ID), token: aliceToken,
		body: map[string]any{"name": "Proyek Alpha v2", "description": "baru"},
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("update project: expected 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	if updated := decodeBody[model.Project](t, rr); updated.Name != "Proyek Alpha v2" {
		t.Errorf("update not applied: %+v", updated)
	}

	rr = doRequest(t, router, requestOptions{method: http.MethodDelete, path: fmt.Sprintf("/projects/%d", project.ID), token: aliceToken})
	if rr.Code != http.StatusNoContent {
		t.Fatalf("delete project: expected 204, got %d", rr.Code)
	}

	rr = doRequest(t, router, requestOptions{method: http.MethodGet, path: fmt.Sprintf("/projects/%d", project.ID), token: aliceToken})
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", rr.Code)
	}
}

func TestProjectRBACAndMembers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	router := newTestRouter(db)

	_, aliceToken := createTestUser(t, db)
	bobID, bobToken := createTestUser(t, db)
	charlieID, charlieToken := createTestUser(t, db)

	rr := doRequest(t, router, requestOptions{
		method: http.MethodPost, path: "/projects", token: aliceToken,
		body: map[string]any{"name": "Project Rahasia Alice"},
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create project: expected 201, got %d", rr.Code)
	}
	project := decodeBody[model.Project](t, rr)

	// Non-member tidak boleh melihat
	rr = doRequest(t, router, requestOptions{method: http.MethodGet, path: fmt.Sprintf("/projects/%d", project.ID), token: bobToken})
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for non-member, got %d", rr.Code)
	}

	// Non-member tidak boleh menambah member
	rr = doRequest(t, router, requestOptions{
		method: http.MethodPost, path: fmt.Sprintf("/projects/%d/members", project.ID), token: bobToken,
		body: map[string]any{"user_id": charlieID, "role": "member"},
	})
	if rr.Code != http.StatusForbidden && rr.Code != http.StatusNotFound {
		t.Errorf("expected 403/404 for non-member add, got %d", rr.Code)
	}

	// Owner menambahkan Bob sebagai member
	rr = doRequest(t, router, requestOptions{
		method: http.MethodPost, path: fmt.Sprintf("/projects/%d/members", project.ID), token: aliceToken,
		body: map[string]any{"user_id": bobID, "role": "member"},
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("owner add member: expected 201, got %d (%s)", rr.Code, rr.Body.String())
	}
	member := decodeBody[model.ProjectMember](t, rr)
	if member.Role != model.RoleMember || member.UserID != bobID {
		t.Errorf("unexpected member: %+v", member)
	}

	// Duplikat member → 409
	rr = doRequest(t, router, requestOptions{
		method: http.MethodPost, path: fmt.Sprintf("/projects/%d/members", project.ID), token: aliceToken,
		body: map[string]any{"user_id": bobID, "role": "member"},
	})
	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate member, got %d", rr.Code)
	}

	// Member sekarang bisa melihat project
	rr = doRequest(t, router, requestOptions{method: http.MethodGet, path: fmt.Sprintf("/projects/%d", project.ID), token: bobToken})
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for member, got %d", rr.Code)
	}

	// Member tidak boleh menambah member lain
	rr = doRequest(t, router, requestOptions{
		method: http.MethodPost, path: fmt.Sprintf("/projects/%d/members", project.ID), token: bobToken,
		body: map[string]any{"user_id": charlieID, "role": "member"},
	})
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 when member adds member, got %d", rr.Code)
	}

	// Owner promosikan Bob menjadi admin
	rr = doRequest(t, router, requestOptions{
		method: http.MethodPatch, path: fmt.Sprintf("/projects/%d/members/%d", project.ID, bobID), token: aliceToken,
		body: map[string]any{"role": "admin"},
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("promote member: expected 200, got %d (%s)", rr.Code, rr.Body.String())
	}

	// Admin tidak boleh menambah admin lain
	rr = doRequest(t, router, requestOptions{
		method: http.MethodPost, path: fmt.Sprintf("/projects/%d/members", project.ID), token: bobToken,
		body: map[string]any{"user_id": charlieID, "role": "admin"},
	})
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 when admin adds admin, got %d", rr.Code)
	}

	// Admin boleh menambah member biasa
	rr = doRequest(t, router, requestOptions{
		method: http.MethodPost, path: fmt.Sprintf("/projects/%d/members", project.ID), token: bobToken,
		body: map[string]any{"user_id": charlieID, "role": "member"},
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("admin add member: expected 201, got %d (%s)", rr.Code, rr.Body.String())
	}

	// Daftar member berisi 3 orang
	rr = doRequest(t, router, requestOptions{method: http.MethodGet, path: fmt.Sprintf("/projects/%d/members", project.ID), token: charlieToken})
	if rr.Code != http.StatusOK {
		t.Fatalf("list members: expected 200, got %d", rr.Code)
	}
	members := decodeBody[struct {
		Data []model.ProjectMember `json:"data"`
	}](t, rr)
	if len(members.Data) != 3 {
		t.Errorf("expected 3 members, got %d", len(members.Data))
	}

	// Member tidak boleh menghapus project
	rr = doRequest(t, router, requestOptions{method: http.MethodDelete, path: fmt.Sprintf("/projects/%d", project.ID), token: charlieToken})
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 when member deletes project, got %d", rr.Code)
	}

	// Admin mengeluarkan member
	rr = doRequest(t, router, requestOptions{
		method: http.MethodDelete, path: fmt.Sprintf("/projects/%d/members", project.ID), token: bobToken,
		body: map[string]any{"user_id": charlieID},
	})
	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204 when admin removes member, got %d (%s)", rr.Code, rr.Body.String())
	}
}

func TestProjectTasksFlow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	router := newTestRouter(db)

	_, aliceToken := createTestUser(t, db)
	bobID, bobToken := createTestUser(t, db)
	_, outsiderToken := createTestUser(t, db)

	rr := doRequest(t, router, requestOptions{
		method: http.MethodPost, path: "/projects", token: aliceToken,
		body: map[string]any{"name": "Project Task"},
	})
	project := decodeBody[model.Project](t, rr)

	// Tambahkan Bob sebagai member agar bisa di-assign
	rr = doRequest(t, router, requestOptions{
		method: http.MethodPost, path: fmt.Sprintf("/projects/%d/members", project.ID), token: aliceToken,
		body: map[string]any{"user_id": bobID, "role": "member"},
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("add member: %d %s", rr.Code, rr.Body.String())
	}

	// Owner membuat task di project, assign ke Bob
	rr = doRequest(t, router, requestOptions{
		method: http.MethodPost, path: fmt.Sprintf("/projects/%d/tasks", project.ID), token: aliceToken,
		body: map[string]any{
			"title": "Task Project", "status": model.StatusTodo, "priority": model.PriorityHigh,
			"deadline": "2026-12-10", "assignee_id": bobID,
		},
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create project task: expected 201, got %d (%s)", rr.Code, rr.Body.String())
	}
	task := decodeBody[model.Task](t, rr)
	if task.ProjectID == nil || *task.ProjectID != project.ID {
		t.Errorf("expected project_id %d, got %v", project.ID, task.ProjectID)
	}

	// Bob (assignee + member) bisa melihat task
	rr = doRequest(t, router, requestOptions{method: http.MethodGet, path: fmt.Sprintf("/tasks/%d", task.ID), token: bobToken})
	if rr.Code != http.StatusOK {
		t.Errorf("expected assignee to read task, got %d", rr.Code)
	}

	// Outsider tidak bisa
	rr = doRequest(t, router, requestOptions{method: http.MethodGet, path: fmt.Sprintf("/tasks/%d", task.ID), token: outsiderToken})
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for outsider, got %d", rr.Code)
	}

	// List task project oleh member
	rr = doRequest(t, router, requestOptions{method: http.MethodGet, path: fmt.Sprintf("/projects/%d/tasks", project.ID), token: bobToken})
	if rr.Code != http.StatusOK {
		t.Fatalf("list project tasks: expected 200, got %d", rr.Code)
	}
	list := decodeBody[listEnvelope[model.Task]](t, rr)
	if len(list.Data) != 1 {
		t.Errorf("expected 1 project task, got %d", len(list.Data))
	}

	// Outsider tidak boleh list
	rr = doRequest(t, router, requestOptions{method: http.MethodGet, path: fmt.Sprintf("/projects/%d/tasks", project.ID), token: outsiderToken})
	if rr.Code != http.StatusForbidden && rr.Code != http.StatusNotFound {
		t.Errorf("expected 403/404 for outsider list, got %d", rr.Code)
	}

	// Assign ke non-member ditolak
	outsiderID := int64(0)
	rrOut := doRequest(t, router, requestOptions{method: http.MethodGet, path: "/me", token: outsiderToken})
	outsider := decodeBody[model.User](t, rrOut)
	outsiderID = outsider.ID

	rr = doRequest(t, router, requestOptions{
		method: http.MethodPost, path: fmt.Sprintf("/projects/%d/tasks", project.ID), token: aliceToken,
		body: map[string]any{
			"title": "Task Invalid", "status": model.StatusTodo, "priority": model.PriorityLow,
			"deadline": "2026-12-10", "assignee_id": outsiderID,
		},
	})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when assigning non-member, got %d", rr.Code)
	}

	// Hapus project → task ikut terhapus (cascade)
	rr = doRequest(t, router, requestOptions{method: http.MethodDelete, path: fmt.Sprintf("/projects/%d", project.ID), token: aliceToken})
	if rr.Code != http.StatusNoContent {
		t.Fatalf("delete project: expected 204, got %d", rr.Code)
	}
	rr = doRequest(t, router, requestOptions{method: http.MethodGet, path: fmt.Sprintf("/tasks/%d", task.ID), token: aliceToken})
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected cascaded task deletion, got %d", rr.Code)
	}
}
