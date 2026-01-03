package events

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/joshp123/xuezh/internal/xuezh/clock"
	"github.com/joshp123/xuezh/internal/xuezh/db"
	"github.com/joshp123/xuezh/internal/xuezh/ids"
	"github.com/joshp123/xuezh/internal/xuezh/jsonio"
	"github.com/joshp123/xuezh/internal/xuezh/paths"
)

type Event struct {
	EventID   string   `json:"event_id"`
	EventType string   `json:"event_type"`
	TS        string   `json:"ts"`
	Modality  string   `json:"modality"`
	Items     []string `json:"items"`
	Context   *string  `json:"context"`
}

func readItemsFile(path string) ([]string, error) {
	resolved, err := paths.ResolveInWorkspace(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	items := []string{}
	for _, line := range lines {
		item := strings.TrimSpace(line)
		if item == "" {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func ParseItems(items, itemsFile string) ([]string, error) {
	parsed := []string{}
	if items != "" {
		for _, part := range strings.Split(items, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				parsed = append(parsed, part)
			}
		}
	}
	if itemsFile != "" {
		fileItems, err := readItemsFile(itemsFile)
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, fileItems...)
	}
	for _, item := range parsed {
		if !ids.IsItemID(item) {
			return nil, fmt.Errorf("invalid item id: %s", item)
		}
	}
	return parsed, nil
}

func LogEvent(eventType, modality string, items []string, context *string) (Event, error) {
	now, err := clock.NowUTC()
	if err != nil {
		return Event{}, err
	}
	eventID := ids.EventIDULID()
	itemsJSON, err := jsonio.Marshal(items)
	if err != nil {
		return Event{}, err
	}
	payloadJSON, err := jsonio.Marshal(map[string]any{})
	if err != nil {
		return Event{}, err
	}
	dbPath, err := db.InitDB()
	if err != nil {
		return Event{}, err
	}
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return Event{}, err
	}
	defer conn.Close()
	_, err = conn.Exec(
		`INSERT INTO events (id, event_type, ts, modality, items_json, context, payload_json)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		eventID, eventType, clock.FormatISO(now), modality, itemsJSON, context, payloadJSON,
	)
	if err != nil {
		return Event{}, err
	}
	return Event{
		EventID:   eventID,
		EventType: eventType,
		TS:        clock.FormatISO(now),
		Modality:  modality,
		Items:     items,
		Context:   context,
	}, nil
}

func ListEvents(since string, limit int) ([]Event, error) {
	now, err := clock.NowUTC()
	if err != nil {
		return nil, err
	}
	duration, err := clock.ParseDuration(since)
	if err != nil {
		return nil, err
	}
	sinceDT := now.Add(-duration)
	dbPath, err := db.InitDB()
	if err != nil {
		return nil, err
	}
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	rows, err := conn.Query(
		`SELECT id, event_type, ts, modality, items_json, context
		 FROM events
		 WHERE ts >= ?
		 ORDER BY ts ASC, id ASC
		 LIMIT ?`,
		clock.FormatISO(sinceDT), limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []Event
	for rows.Next() {
		var id, eventType, ts, modality, itemsJSON string
		var context sql.NullString
		if err := rows.Scan(&id, &eventType, &ts, &modality, &itemsJSON, &context); err != nil {
			return nil, err
		}
		items := []string{}
		if itemsJSON != "" {
			if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
				return nil, err
			}
		}
		var ctxPtr *string
		if context.Valid {
			ctx := context.String
			ctxPtr = &ctx
		}
		results = append(results, Event{
			EventID:   id,
			EventType: eventType,
			TS:        ts,
			Modality:  modality,
			Items:     items,
			Context:   ctxPtr,
		})
	}
	return results, rows.Err()
}

func ExposureCounts(since string) (map[string]int, error) {
	now, err := clock.NowUTC()
	if err != nil {
		return nil, err
	}
	duration, err := clock.ParseDuration(since)
	if err != nil {
		return nil, err
	}
	sinceDT := now.Add(-duration)
	dbPath, err := db.InitDB()
	if err != nil {
		return nil, err
	}
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	rows, err := conn.Query(
		`SELECT modality, COUNT(*)
		 FROM events
		 WHERE event_type = 'exposure' AND ts >= ?
		 GROUP BY modality`,
		clock.FormatISO(sinceDT),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var modality string
		var count int
		if err := rows.Scan(&modality, &count); err != nil {
			return nil, err
		}
		counts[modality] = count
	}
	return counts, rows.Err()
}

// no home expansion needed; workspace resolution handles paths
