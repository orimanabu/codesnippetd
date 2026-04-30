package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddleware_AddsHeaders(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	corsMiddleware(next).ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin: got %q, want %q", got, "*")
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, OPTIONS" {
		t.Errorf("Access-Control-Allow-Methods: got %q, want %q", got, "GET, POST, OPTIONS")
	}
	if got := w.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type" {
		t.Errorf("Access-Control-Allow-Headers: got %q, want %q", got, "Content-Type")
	}
}

func TestCORSMiddleware_OptionsPreflightShortCircuits(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusTeapot)
	})

	req := httptest.NewRequest(http.MethodOptions, "/tags", nil)
	w := httptest.NewRecorder()

	corsMiddleware(next).ServeHTTP(w, req)

	if called {
		t.Fatal("expected preflight request to bypass next handler")
	}
	if w.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusNoContent)
	}
}
