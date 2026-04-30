package webserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joshp123/xuezh/internal/xuezh/hellochinese"
	"github.com/joshp123/xuezh/internal/xuezh/paths"
)

type ServerOptions struct {
	Port int
}

func Serve(opts ServerOptions) error {
	port := opts.Port
	if port == 0 {
		port = 8765
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/cram/next", handleNext)
	mux.HandleFunc("POST /api/cram/grade", handleGrade)
	mux.HandleFunc("GET /artifacts/", handleArtifact)
	mux.Handle("/", staticHandler())
	return http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", port), mux)
}

func handleNext(w http.ResponseWriter, r *http.Request) {
	cards, err := hellochinese.NextCards(1, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if len(cards) == 0 {
		writeJSON(w, map[string]any{"card": nil})
		return
	}
	writeJSON(w, map[string]any{"card": cards[0]})
}

func handleGrade(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ItemID string `json:"item_id"`
		Grade  string `json:"grade"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	result, err := hellochinese.GradeCard(req.ItemID, req.Grade, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, result)
}

func handleArtifact(w http.ResponseWriter, r *http.Request) {
	rel := strings.TrimPrefix(r.URL.Path, "/")
	resolved, err := paths.ResolveInWorkspace(rel)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	http.ServeFile(w, r, resolved)
}

func staticHandler() http.Handler {
	dist := filepath.Join("web", "dist")
	if info, err := os.Stat(dist); err == nil && info.IsDir() {
		return http.FileServer(http.Dir(dist))
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<main style="font:16px system-ui;padding:32px;max-width:720px;margin:auto"><h1>xuezh web assets not built</h1><p>Run <code>cd web && pnpm install && pnpm build</code>, then restart <code>xuezh web serve</code>.</p></main>`))
	})
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
}
