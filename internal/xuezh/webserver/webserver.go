package webserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joshp123/xuezh/internal/xuezh/cram"
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
	mux.HandleFunc("GET /api/cram/overview", handleOverview)
	mux.HandleFunc("POST /api/cram/preview", handlePreview)
	mux.HandleFunc("GET /api/cram/session", handleActiveSession)
	mux.HandleFunc("POST /api/cram/session/start", handleStartSession)
	mux.HandleFunc("POST /api/cram/session/reveal", handleRevealSession)
	mux.HandleFunc("POST /api/cram/session/repeat", handleRepeatSession)
	mux.HandleFunc("POST /api/cram/session/grade", handleSessionGrade)
	mux.HandleFunc("POST /api/cram/session/undo", handleSessionUndo)
	mux.HandleFunc("GET /api/cram/offline/deck", handleOfflineDeck)
	mux.HandleFunc("POST /api/cram/offline/sync", handleOfflineSync)
	mux.HandleFunc("GET /offline/app-shell", handleOfflineAppShell)
	mux.HandleFunc("GET /artifacts/", handleArtifact)
	mux.Handle("/", staticHandler())
	return http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", port), mux)
}

func handleOverview(w http.ResponseWriter, r *http.Request) {
	overview, err := cram.OverviewFor(time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, overview)
}

func handlePreview(w http.ResponseWriter, r *http.Request) {
	var req cram.PracticeFilters
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	preview, err := cram.PracticePreviewFor(req, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, preview)
}

func handleActiveSession(w http.ResponseWriter, r *http.Request) {
	session, err := cram.ActiveReviewSession()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, map[string]any{"session": session})
}

func handleStartSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Limit   int      `json:"limit"`
		CardIDs []string `json:"card_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	session, err := cram.StartReviewSession(cram.ReviewSessionStartOptions{Limit: req.Limit, CardIDs: req.CardIDs}, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, map[string]any{"session": session})
}

func handleRevealSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	session, err := cram.RevealReviewSession(req.SessionID, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, map[string]any{"session": session})
}

func handleRepeatSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	session, err := cram.ToggleReviewSessionRepeat(req.SessionID, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, map[string]any{"session": session})
}

func handleSessionGrade(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ItemID     string `json:"item_id"`
		Grade      string `json:"grade"`
		SessionID  string `json:"session_id"`
		ShownAt    string `json:"shown_at"`
		AnsweredAt string `json:"answered_at"`
		ElapsedMS  int    `json:"elapsed_ms"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	session, result, err := cram.GradeReviewSession(cram.GradeOptions{
		ItemID:     req.ItemID,
		Grade:      req.Grade,
		SessionID:  req.SessionID,
		ShownAt:    req.ShownAt,
		AnsweredAt: req.AnsweredAt,
		ElapsedMS:  req.ElapsedMS,
	}, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, map[string]any{"session": session, "grade": result})
}

func handleSessionUndo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	session, result, err := cram.UndoReviewSession(req.SessionID, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, map[string]any{"session": session, "undo": result})
}

func handleOfflineDeck(w http.ResponseWriter, r *http.Request) {
	deck, err := cram.OfflineDeck(time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, deck)
}

func handleOfflineSync(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Events []cram.OfflineReviewEvent `json:"events"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	result, err := cram.SyncOfflineReviewEvents(req.Events, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, result)
}

func handleOfflineAppShell(w http.ResponseWriter, r *http.Request) {
	assets, err := offlineAppShellAssets()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, map[string]any{"assets": assets})
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
		files := http.FileServer(http.Dir(dist))
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeNoStore(w)
			if r.URL.Path == "/xuezh" {
				http.ServeFile(w, r, filepath.Join(dist, "index.html"))
				return
			}
			files.ServeHTTP(w, r)
		})
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeNoStore(w)
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<main style="font:16px system-ui;padding:32px;max-width:720px;margin:auto"><h1>xuezh web assets not built</h1><p>Run <code>cd web && pnpm install && pnpm build</code>, then restart <code>xuezh web serve</code>.</p></main>`))
	})
}

func offlineAppShellAssets() ([]string, error) {
	dist := filepath.Join("web", "dist")
	assets := []string{"/", "/xuezh", "/index.html", "/manifest.webmanifest", "/sw.js"}
	assetDir := filepath.Join(dist, "assets")
	entries, err := os.ReadDir(assetDir)
	if err != nil {
		if os.IsNotExist(err) {
			return assets, nil
		}
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		assets = append(assets, "/assets/"+entry.Name())
	}
	return assets, nil
}

func writeNoStore(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
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
