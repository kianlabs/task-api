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

	// Register
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

	// Duplikat email
	rr = doRequest(t, router, requestOptions{
		method: http.MethodPost, path: "/register",
		body: map[string]any{"email": email, "password": password},
	})
	if rr.Code != http.StatusConflict {
		t.Errorf("duplicate register: expected 409, got %d", rr.Code)
	}

	// Password salah
	rr = doRequest(t, router, requestOptions{
		method: http.MethodPost, path: "/login",
		body: map[string]any{"email": email, "password": "salahbanget"},
	})
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("wrong password: expected 401, got %d", rr.Code)
	}

	// Login benar
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

	// Profil
	rr = doRequest(t, router, requestOptions{method: http.MethodGet, path: "/me", token: token})
	if rr.Code != http.StatusOK {
		t.Fatalf("get profile: expected 200, got %d", rr.Code)
	}

	// Update email
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

	// Ganti password: current salah
	rr = doRequest(t, router, requestOptions{
		method: http.MethodPatch, path: "/me/password", token: token,
		body: map[string]any{"current_password": "salah", "new_password": "newsecret123"},
	})
	if rr.Code != http.StatusForbidden {
		t.Errorf("wrong current password: expected 403, got %d", rr.Code)
	}

	// Ganti password: benar
	rr = doRequest(t, router, requestOptions{
		method: http.MethodPatch, path: "/me/password", token: token,
		body: map[string]any{"current_password": password, "new_password": "newsecret123"},
	})
	if rr.Code != http.StatusNoContent {
		t.Fatalf("change password: expected 204, got %d (%s)", rr.Code, rr.Body.String())
	}

	// Password lama harus gagal
	rr = doRequest(t, router, requestOptions{
		method: http.MethodPost, path: "/login",
		body: map[string]any{"email": newEmail, "password": password},
	})
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("old password: expected 401, got %d", rr.Code)
	}

	// Password baru harus berhasil
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

	// Email yang sama dengan kapitalisasi berbeda harus ditolak.
	rr = doRequest(t, router, requestOptions{
		method: http.MethodPost, path: "/register",
		body: map[string]any{"email": fmt.Sprintf("case_%d@example.com", suffix), "password": "secret123"},
	})
	if rr.Code != http.StatusConflict {
		t.Errorf("case-insensitive duplicate: expected 409, got %d (%s)", rr.Code, rr.Body.String())
	}
}
