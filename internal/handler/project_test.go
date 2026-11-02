package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"task-api/internal/auth"
	"task-api/internal/model"
	"task-api/internal/repository"
	"task-api/internal/service"
	"testing"
	"time"
)

func TestProjectOwnership_CrossUserAccess(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	projectRepo := repository.NewProjectRepository(db)
	projectService := service.NewProjectService(projectRepo)
	h := NewProjectHandler(projectService)

	mux := http.NewServeMux()
	authMiddleware := auth.Middleware(testJWTSecret)
	mux.Handle("POST /projects", authMiddleware(http.HandlerFunc(h.CreateProject)))
	mux.Handle("GET /projects", authMiddleware(http.HandlerFunc(h.GetProjects)))
	mux.Handle("GET /projects/{id}", authMiddleware(http.HandlerFunc(h.GetProjectByID)))
	mux.Handle("POST /projects/{id}/members", authMiddleware(http.HandlerFunc(h.AddMember)))

	_, aliceToken := createTestUser(t, db, fmt.Sprintf("alice_%d@example.com", time.Now().UnixNano()))
	_, bobToken := createTestUser(t, db, fmt.Sprintf("bob_%d@example.com", time.Now().UnixNano()))
	// Alice membuat project
	createBody := `{"name":"Project Rahasia Alice","description":"Top Secret"}`
	createReq := httptest.NewRequest(http.MethodPost, "/projects", strings.NewReader(createBody))
	createReq.Header.Set("Authorization", "Bearer "+aliceToken)
	createReq.Header.Set("Content-Type", "application/json")
	createRr := httptest.NewRecorder()
	mux.ServeHTTP(createRr, createReq)

	if createRr.Code != http.StatusCreated {
		t.Fatalf("expected status 201 for Alice create project, got %d", createRr.Code)
	}

	var createdProject model.Project
	if err := json.Unmarshal(createRr.Body.Bytes(), &createdProject); err != nil {
		t.Fatalf("failed to decode created project: %v", err)
	}

	// Bob mencoba mengakses project Alice (Harus 404 Not Found)
	bobReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/projects/%d", createdProject.ID), nil)
	bobReq.Header.Set("Authorization", "Bearer "+bobToken)
	bobRr := httptest.NewRecorder()
	mux.ServeHTTP(bobRr, bobReq)

	if bobRr.Code != http.StatusNotFound {
		t.Errorf("expected 404 when Bob accesses Alice's project, got %d", bobRr.Code)
	}

	// Bob melihat daftar project miliknya (Harus kosong 0 item)
	bobListReq := httptest.NewRequest(http.MethodGet, "/projects", nil)
	bobListReq.Header.Set("Authorization", "Bearer "+bobToken)
	bobListRr := httptest.NewRecorder()
	mux.ServeHTTP(bobListRr, bobListReq)

	if bobListRr.Code != http.StatusOK {
		t.Errorf("expected 200 for Bob get projects, got %d", bobListRr.Code)
	}

	var bobProjects []model.Project
	if err := json.Unmarshal(bobListRr.Body.Bytes(), &bobProjects); err != nil {
		t.Fatalf("failed to decode Bob projects: %v", err)
	}

	if len(bobProjects) != 0 {
		t.Errorf("expected Bob to have 0 projects, got %d", len(bobProjects))
	}

	// Buat user ketiga: Charlie
	charlieID, _ := createTestUser(t, db, fmt.Sprintf("charlie_%d@example.com", time.Now().UnixNano()))

	// Bob (Bukan Owner) mencoba menambahkan Charlie ke project Alice -> Wajib 403 Forbidden
	addMemberBobBody := fmt.Sprintf(`{"user_id":%d,"role":"member"}`, charlieID)
	addMemberBobReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/projects/%d/members", createdProject.ID), strings.NewReader(addMemberBobBody))
	addMemberBobReq.Header.Set("Authorization", "Bearer "+bobToken)
	addMemberBobReq.Header.Set("Content-Type", "application/json")
	addMemberBobRr := httptest.NewRecorder()
	mux.ServeHTTP(addMemberBobRr, addMemberBobReq)

	if addMemberBobRr.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden when non-owner Bob adds member, got %d", addMemberBobRr.Code)
	}

	// Alice (Owner Sah) menambahkan Charlie ke projectnya -> Wajib 201 Created
	addMemberAliceBody := fmt.Sprintf(`{"user_id":%d,"role":"member"}`, charlieID)
	addMemberAliceReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/projects/%d/members", createdProject.ID), strings.NewReader(addMemberAliceBody))
	addMemberAliceReq.Header.Set("Authorization", "Bearer "+aliceToken)
	addMemberAliceReq.Header.Set("Content-Type", "application/json")
	addMemberAliceRr := httptest.NewRecorder()
	mux.ServeHTTP(addMemberAliceRr, addMemberAliceReq)

	if addMemberAliceRr.Code != http.StatusCreated {
		t.Errorf("expected 201 Created when owner Alice adds member, got %d", addMemberAliceRr.Code)
	}
}
