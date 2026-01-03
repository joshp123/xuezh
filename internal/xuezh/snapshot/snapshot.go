package snapshot

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/joshp123/xuezh/internal/xuezh/clock"
	"github.com/joshp123/xuezh/internal/xuezh/db"
	"github.com/joshp123/xuezh/internal/xuezh/envelope"
	"github.com/joshp123/xuezh/internal/xuezh/events"
	"github.com/joshp123/xuezh/internal/xuezh/jsonio"
	"github.com/joshp123/xuezh/internal/xuezh/paths"
)

type Result struct {
	Data      map[string]any
	Artifacts []envelope.Artifact
	Truncated bool
	Limits    map[string]any
}

func BuildSnapshot(window string, dueLimit int, evidenceLimit int, maxBytes int) (Result, error) {
	now, err := clock.NowUTC()
	if err != nil {
		return Result{}, err
	}
	workspace, err := paths.EnsureWorkspace()
	if err != nil {
		return Result{}, err
	}
	dbPath, err := db.InitDB()
	if err != nil {
		return Result{}, err
	}
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return Result{}, err
	}
	defer conn.Close()

	hskSummary := map[string]map[string]any{}
	countsByLevel := map[string]map[string]any{}
	for _, entry := range []struct {
		datasetType string
		key         string
	}{
		{datasetType: "hsk_vocab", key: "vocab"},
		{datasetType: "hsk_grammar", key: "grammar"},
		{datasetType: "hsk_chars", key: "chars"},
	} {
		datasetID, err := latestDatasetID(conn, entry.datasetType)
		if err != nil {
			return Result{}, err
		}
		if datasetID == "" {
			continue
		}
		counts, err := countsByLevelForDataset(conn, datasetID)
		if err != nil {
			return Result{}, err
		}
		countsByLevel[entry.key] = counts
		total := 0
		for _, value := range counts {
			if n, ok := value.(int); ok {
				total += n
			}
		}
		hskSummary[entry.key] = map[string]any{"total": total}
	}

	recentEvents, err := events.ListEvents(window, evidenceLimit)
	if err != nil {
		return Result{}, err
	}
	exposureCounts, err := events.ExposureCounts(window)
	if err != nil {
		return Result{}, err
	}
	recentPayload := []map[string]any{}
	for _, event := range recentEvents {
		recentPayload = append(recentPayload, map[string]any{
			"event_id":   event.EventID,
			"event_type": event.EventType,
			"ts":         event.TS,
			"modality":   event.Modality,
			"items":      event.Items,
			"context":    event.Context,
		})
	}

	data := map[string]any{
		"generated_at":      clock.FormatISO(now),
		"window":            window,
		"recent_events":     recentPayload,
		"exposure_counts":   exposureCounts,
		"due_items":         []any{},
		"due_counts_by_day": map[string]any{},
		"hsk_summary":       hskSummary,
		"counts_by_level":   countsByLevel,
		"limits": map[string]any{
			"due_limit":      dueLimit,
			"evidence_limit": evidenceLimit,
		},
	}

	envelopeData := map[string]any{
		"ok":             true,
		"schema_version": "1",
		"command":        "snapshot",
		"data":           data,
		"artifacts":      []envelope.Artifact{},
		"truncated":      false,
		"limits":         map[string]any{"max_bytes": maxBytes},
	}
	payload, err := jsonio.Dumps(envelopeData)
	if err != nil {
		return Result{}, err
	}
	if len([]byte(payload)) <= maxBytes {
		return Result{Data: data, Artifacts: []envelope.Artifact{}, Truncated: false, Limits: map[string]any{"max_bytes": maxBytes}}, nil
	}

	spillPath, err := artifactPath("snapshot", now)
	if err != nil {
		return Result{}, err
	}
	if err := os.WriteFile(spillPath, []byte(payload), 0o644); err != nil {
		return Result{}, err
	}
	rel, err := filepath.Rel(workspace, spillPath)
	if err != nil {
		return Result{}, err
	}
	stat, err := os.Stat(spillPath)
	if err != nil {
		return Result{}, err
	}
	artifact := envelope.Artifact{Path: rel, MIME: "application/json", Purpose: "snapshot_spill", Bytes: intPtr(int(stat.Size()))}
	truncatedData := map[string]any{
		"generated_at":    clock.FormatISO(now),
		"window":          window,
		"recent_events":   []any{},
		"exposure_counts": map[string]any{},
		"spill_artifact":  rel,
	}
	return Result{Data: truncatedData, Artifacts: []envelope.Artifact{artifact}, Truncated: true, Limits: map[string]any{"max_bytes": maxBytes}}, nil
}

func latestDatasetID(conn *sql.DB, datasetType string) (string, error) {
	row := conn.QueryRow(
		"SELECT id FROM datasets WHERE dataset_type = ? ORDER BY ingested_at DESC LIMIT 1",
		datasetType,
	)
	var id string
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return id, nil
}

func countsByLevelForDataset(conn *sql.DB, datasetID string) (map[string]any, error) {
	rows, err := conn.Query("SELECT payload_json FROM dataset_items WHERE dataset_id = ?", datasetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := map[string]any{}
	for rows.Next() {
		var payloadJSON string
		if err := rows.Scan(&payloadJSON); err != nil {
			return nil, err
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
			return nil, err
		}
		level := fmt.Sprintf("%v", payload["hsk_level"])
		if level == "<nil>" {
			level = "None"
		}
		value, _ := counts[level].(int)
		counts[level] = value + 1
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return counts, nil
}

func artifactPath(prefix string, now time.Time) (string, error) {
	root, err := paths.EnsureWorkspace()
	if err != nil {
		return "", err
	}
	dayPath := filepath.Join(root, "artifacts", now.Format("2006"), now.Format("01"), now.Format("02"))
	if err := os.MkdirAll(dayPath, 0o755); err != nil {
		return "", err
	}
	filename := fmt.Sprintf("%s-%s.json", prefix, now.UTC().Format("20060102T150405Z"))
	return filepath.Join(dayPath, filename), nil
}

func intPtr(value int) *int {
	return &value
}
