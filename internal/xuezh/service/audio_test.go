package service

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/joshp123/xuezh/internal/xuezh/envelope"
)

func TestAppRecordsPronunciationAttempt(t *testing.T) {
	useServiceTestWorkspace(t)
	now := time.Date(2026, 5, 28, 9, 10, 11, 0, time.UTC)
	artifacts := []envelope.Artifact{{
		Path:    "artifacts/assessment.json",
		MIME:    "application/json",
		Purpose: "assessment",
	}}
	summary := map[string]any{"assessment": map[string]any{"exact_match": true}}

	attemptID, err := New().RecordPronunciationAttempt("local", artifacts, summary, now)
	if err != nil {
		t.Fatal(err)
	}
	if attemptID == "" {
		t.Fatal("empty attempt id")
	}

	conn := openServiceTestDB(t)
	defer conn.Close()
	var backend, ts, artifactsJSON, summaryJSON string
	if err := conn.QueryRow(
		`SELECT backend_id, ts, artifacts_json, summary_json FROM pronunciation_attempts WHERE id = ?`,
		attemptID,
	).Scan(&backend, &ts, &artifactsJSON, &summaryJSON); err != nil {
		if err == sql.ErrNoRows {
			t.Fatal("pronunciation attempt was not recorded")
		}
		t.Fatal(err)
	}
	if backend != "local" || ts != "2026-05-28T09:10:11+00:00" || !strings.Contains(artifactsJSON, "assessment") || !strings.Contains(summaryJSON, "exact_match") {
		t.Fatalf("unexpected attempt row: backend=%q ts=%q artifacts=%s summary=%s", backend, ts, artifactsJSON, summaryJSON)
	}
}
