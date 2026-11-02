package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimitAllowsBurstThenBlocks(t *testing.T) {
	handler := RateLimit(0.001, 2)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	call := func() int {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = "203.0.113.5:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		return rr.Code
	}

	if code := call(); code != http.StatusOK {
		t.Fatalf("request 1: expected 200, got %d", code)
	}
	if code := call(); code != http.StatusOK {
		t.Fatalf("request 2: expected 200, got %d", code)
	}
	if code := call(); code != http.StatusTooManyRequests {
		t.Fatalf("request 3: expected 429, got %d", code)
	}
}

func TestRateLimitSeparateIPs(t *testing.T) {
	handler := RateLimit(0.001, 1)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	call := func(ip string) int {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = ip + ":1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		return rr.Code
	}

	if code := call("198.51.100.1"); code != http.StatusOK {
		t.Fatalf("ip1 first: expected 200, got %d", code)
	}
	if code := call("198.51.100.2"); code != http.StatusOK {
		t.Fatalf("ip2 first: expected 200, got %d", code)
	}
	if code := call("198.51.100.1"); code != http.StatusTooManyRequests {
		t.Fatalf("ip1 second: expected 429, got %d", code)
	}
}
