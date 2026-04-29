package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// registerHandlers registers all application routes on mux.
func registerHandlers(mux *http.ServeMux, useTreeSitter bool) {
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	mux.HandleFunc("POST /pipe", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to read request body: %v", err), http.StatusInternalServerError)
			return
		}
		pipe.mu.Lock()
		if r.URL.Query().Get("mode") == "append" {
			pipe.data = append(pipe.data, body...)
		} else {
			pipe.data = body
		}
		pipe.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("GET /pipe/status", func(w http.ResponseWriter, r *http.Request) {
		pipe.mu.Lock()
		empty := len(pipe.data) == 0
		pipe.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if empty {
			fmt.Fprintln(w, `{"empty":true}`)
		} else {
			fmt.Fprintln(w, `{"empty":false}`)
		}
	})

	mux.HandleFunc("GET /pipe", func(w http.ResponseWriter, r *http.Request) {
		pipe.mu.Lock()
		data := pipe.data
		pipe.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

	mux.HandleFunc("GET /tags", func(w http.ResponseWriter, r *http.Request) {
		tagsPath, err := queryTagsPath(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		db, err := loadTagsFile(tagsPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				http.Error(w, fmt.Sprintf("tags file not found: %s", tagsPath), http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("failed to load tags file: %v", err), http.StatusInternalServerError)
			}
			return
		}

		// Collect all tags from the database
		var all []Tag
		for _, tags := range db.tags {
			all = append(all, tags...)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(all); err != nil {
			log.Printf("encoding response: %v", err)
		}
	})

	mux.HandleFunc("GET /tags/{name}", func(w http.ResponseWriter, r *http.Request) {
		tagName := r.PathValue("name")
		tagsPath, err := queryTagsPath(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		results, err := lookupTag(tagsPath, tagName)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				http.Error(w, fmt.Sprintf("tags file not found: %s", tagsPath), http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("readtags error: %v", err), http.StatusInternalServerError)
			}
			return
		}

		if len(results) == 0 {
			http.Error(w, fmt.Sprintf("tag not found: %s", tagName), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(results); err != nil {
			log.Printf("encoding response: %v", err)
		}
	})

	mux.HandleFunc("GET /snippets/{name}", func(w http.ResponseWriter, r *http.Request) {
		tagName := r.PathValue("name")
		tagsPath, err := queryTagsPath(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		contextDir := filepath.Dir(tagsPath)

		results, err := lookupTag(tagsPath, tagName)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				http.Error(w, fmt.Sprintf("tags file not found: %s", tagsPath), http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("readtags error: %v", err), http.StatusInternalServerError)
			}
			return
		}

		if len(results) == 0 {
			http.Error(w, fmt.Sprintf("tag not found: %s", tagName), http.StatusNotFound)
			return
		}

		var snippets []Snippet
		for _, tag := range results {
			s, err := snippetForTag(r.Context(), tag, contextDir, useTreeSitter)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			snippets = append(snippets, s)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(snippets); err != nil {
			log.Printf("encoding response: %v", err)
		}
	})

	mux.HandleFunc("GET /lines/{name}", func(w http.ResponseWriter, r *http.Request) {
		tagName := r.PathValue("name")
		tagsPath, err := queryTagsPath(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		contextDir := filepath.Dir(tagsPath)

		results, err := lookupTag(tagsPath, tagName)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				http.Error(w, fmt.Sprintf("tags file not found: %s", tagsPath), http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("readtags error: %v", err), http.StatusInternalServerError)
			}
			return
		}

		if len(results) == 0 {
			http.Error(w, fmt.Sprintf("tag not found: %s", tagName), http.StatusNotFound)
			return
		}

		var ranges []LineRange
		for _, tag := range results {
			lr, err := lineRangeForTag(r.Context(), tag, contextDir, useTreeSitter)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			ranges = append(ranges, lr)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(ranges); err != nil {
			log.Printf("encoding response: %v", err)
		}
	})
}
