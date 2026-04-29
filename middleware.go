package main

import (
	"context"
	"log"
	"net/http"
	"time"
)

// corsMiddleware sets Access-Control-Allow-Origin on every response and handles
// preflight OPTIONS requests so that browser clients (e.g. canvas.html loaded
// from a file:// origin) are not blocked by CORS policy.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// accessLog is middleware that logs each request's method, path, status code, and duration.
// It also logs whether tree-sitter was used to resolve an end line.
func accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		meta := &requestMeta{}
		r = r.WithContext(context.WithValue(r.Context(), requestMetaKey{}, meta))
		rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		query := r.URL.RawQuery
		if r.URL.Path == "/pipe" {
			log.Printf("%s %s query=%q %d %s", r.Method, r.URL.Path, query, rec.status, time.Since(start))
		} else {
			parser := "ctags"
			if meta.usedTreeSitter {
				parser = "tree-sitter"
			}
			log.Printf("%s %s query=%q %d %s parser=%s", r.Method, r.URL.Path, query, rec.status, time.Since(start), parser)
		}
	})
}
