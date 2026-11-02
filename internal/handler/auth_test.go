package handler

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"task-api/internal/model"
)

func TestAuthFlow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	router := newTestRouter(db)

	email := fmt.Sprintf("auth_%d@example.com", time.Now().UnixNano())
	password := "secret123"

	rr := doRequest(t, router, requestOptions{
		method: http.MethodPost, path: "/register",
		body: map[string]any{"email": email, "password": password},
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d (%s)", rr.Code, rr.Body.String())
	}
	user := decodeBody[model.User](t, rr)
	if user.ID == 0 || user.Email != email {
		t.Errorf("unexpected user: %+v", user)
	}

	rr = doRequest(t, router, requestOptions{
		method: http.MethodPost, path: "/register",
		body: map[string]any{"email": email, "password": password},
	})
	if rr.Code != http.StatusConflict {
		t.Errorf("duplicate register: expected 409, got %d", rr.Code)
	}

	rr = doRequest(t, router, requestOptions{
		method: http.MethodPost, path: "/login",
		body: map[string]any{"email": email, "password": "salahbanget"},
	})
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("wrong password: expected 401, got %d", rr.Code)
	}

	rr = doRequest(t, router, requestOptions{
		method: http.MethodPost, path: "/login",
		body: map[string]any{"email": email, "password": password},
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	login := decodeBody[struct {
		Token string     `json:"token"`
		User  model.User `json:"user"`
	}](t, rr)
	if login.Token == "" {
		t.Fatal("expected token")
	}
	token := login.Token

	rr = doRequest(t, router, requestOptions{method: http.MethodGet, path: "/me", token: token})
	if rr.Code != http.StatusOK {
		t.Fatalf("get profile: expected 200, got %d", rr.Code)
	}

	newEmail := fmt.Sprintf("auth_new_%d@example.com", time.Now().UnixNano())
	rr = doRequest(t, router, requestOptions{
		method: http.MethodPatch, path: "/me", token: token,
		body: map[string]any{"email": newEmail},
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("update profile: expected 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	if updated := decodeBody[model.User](t, rr); updated.Email != newEmail {
		t.Errorf("email not updated: %+v", updated)
	}

	rr = doRequest(t, router, requestOptions{
		method: http.MethodPatch, path: "/me/password", token: token,
		body: map[string]any{"current_password": "salah", "new_password": "newsecret123"},
	})
	if rr.Code != http.StatusForbidden {
		t.Errorf("wrong current password: expected 403, got %d", rr.Code)
	}

	rr = doRequest(t, router, requestOptions{
		method: http.MethodPatch, path: "/me/password", token: token,
		body: map[string]any{"current_password": password, "new_password": "newsecret123"},
	})
	if rr.Code != http.StatusNoContent {
		t.Fatalf("change password: expected 204, got %d (%s)", rr.Code, rr.Body.String())
	}

	rr = doRequest(t, router, requestOptions{
		method: http.MethodPost, path: "/login",
		body: map[string]any{"email": newEmail, "password": password},
	})
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("old password: expected 401, got %d", rr.Code)
	}

	rr = doRequest(t, router, requestOptions{
		method: http.MethodPost, path: "/login",
		body: map[string]any{"email": newEmail, "password": "newsecret123"},
	})
	if rr.Code != http.StatusOK {
		t.Errorf("new password: expected 200, got %d (%s)", rr.Code, rr.Body.String())
	}
}

func TestUniqueEmailCaseInsensitive(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	router := newTestRouter(db)

	suffix := time.Now().UnixNano()
	first := fmt.Sprintf("Case_%d@Example.com", suffix)

	rr := doRequest(t, router, requestOptions{
		method: http.MethodPost, path: "/register",
		body: map[string]any{"email": first, "password": "secret123"},
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d (%s)", rr.Code, rr.Body.String())
	}

	rr = doRequest(t, router, requestOptions{
		method: http.MethodPost, path: "/register",
		body: map[string]any{"email": fmt.Sprintf("case_%d@example.com", suffix), "password": "secret123"},
	})
	if rr.Code != http.StatusConflict {
		t.Errorf("case-insensitive duplicate: expected 409, got %d (%s)", rr.Code, rr.Body.String())
	}
}
