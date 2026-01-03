package srs

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/joshp123/xuezh/internal/xuezh/clock"
	"github.com/joshp123/xuezh/internal/xuezh/db"
	"github.com/joshp123/xuezh/internal/xuezh/ids"
	"github.com/joshp123/xuezh/internal/xuezh/jsonio"
)

type DueItem struct {
	ItemID     string `json:"item_id"`
	DueAt      string `json:"due_at"`
	ReviewType string `json:"review_type"`
}

func intervalDays(rule string, grade int) int {
	table := map[int]int{0: 1, 1: 1, 2: 2, 3: 4, 4: 7, 5: 14}
	if value, ok := table[grade]; ok {
		return value
	}
	return 1
}

func ScheduleNextDue(grade int, now time.Time, rule, nextDue string) (string, string, error) {
	if nextDue != "" {
		dt, err := clock.ParseUTCISO(nextDue)
		if err != nil {
			return "", "", err
		}
		return clock.FormatISO(dt), "", nil
	}
	appliedRule := rule
	if appliedRule == "" {
		appliedRule = "sm2"
	}
	days := intervalDays(appliedRule, grade)
	dueAt := now.Add(time.Duration(days) * 24 * time.Hour)
	return clock.FormatISO(dueAt), appliedRule, nil
}

func dueExpr(reviewType string) (string, error) {
	if reviewType == "recall" {
		return "COALESCE(recall_due_at, due_at)", nil
	}
	if reviewType == "pronunciation" {
		return "pronunciation_due_at", nil
	}
	return "", fmt.Errorf("unsupported review type: %s", reviewType)
}

func ListDueItems(limit int, now time.Time, reviewType string) ([]DueItem, error) {
	de, err := dueExpr(reviewType)
	if err != nil {
		return nil, err
	}
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
		fmt.Sprintf(`
		SELECT item_id, %s
		FROM user_knowledge
		WHERE %s IS NOT NULL AND %s <= ?
		ORDER BY %s ASC, item_id ASC
		LIMIT ?`, de, de, de, de),
		clock.FormatISO(now), limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []DueItem{}
	for rows.Next() {
		var itemID, dueAt string
		if err := rows.Scan(&itemID, &dueAt); err != nil {
			return nil, err
		}
		items = append(items, DueItem{ItemID: itemID, DueAt: dueAt, ReviewType: reviewType})
	}
	return items, rows.Err()
}

func RecordReviewEvent(itemID, eventType string, payload map[string]any, now time.Time) error {
	dbPath, err := db.InitDB()
	if err != nil {
		return err
	}
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer conn.Close()
	eventID := ids.EventIDULID()
	payloadJSON, err := jsonio.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = conn.Exec(
		`INSERT INTO review_events (id, item_id, event_type, ts, session_id, payload_json)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		eventID, itemID, eventType, clock.FormatISO(now), nil, payloadJSON,
	)
	return err
}

func UpsertKnowledge(itemID string, recallDueAt *string, recallGrade *int, pronunciationDueAt *string, pronunciationGrade *int, now time.Time) error {
	if recallDueAt == nil && pronunciationDueAt == nil {
		return nil
	}
	dbPath, err := db.InitDB()
	if err != nil {
		return err
	}
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer conn.Close()
	itemType := ids.ItemType(itemID)
	if itemType == "" {
		itemType = "unknown"
	}
	var seenCount int
	err = conn.QueryRow("SELECT seen_count FROM user_knowledge WHERE item_id = ?", itemID).Scan(&seenCount)
	if err == nil {
		seenCount++
		updates := []string{}
		params := []any{}
		if recallDueAt != nil {
			updates = append(updates, "recall_due_at = ?", "recall_last_grade = ?", "due_at = ?", "last_grade = ?")
			params = append(params, *recallDueAt, recallGrade, *recallDueAt, recallGrade)
		}
		if pronunciationDueAt != nil {
			updates = append(updates, "pronunciation_due_at = ?", "pronunciation_last_grade = ?")
			params = append(params, *pronunciationDueAt, pronunciationGrade)
		}
		updates = append(updates, "last_seen_at = ?", "seen_count = ?")
		params = append(params, clock.FormatISO(now), seenCount, itemID)
		query := "UPDATE user_knowledge SET " + strings.Join(updates, ", ") + " WHERE item_id = ?"
		_, err = conn.Exec(query, params...)
		return err
	}
	if err != sql.ErrNoRows {
		return err
	}
	_, err = conn.Exec(
		`INSERT INTO user_knowledge
		(
		  item_id,
		  item_type,
		  modality,
		  first_seen_at,
		  last_seen_at,
		  seen_count,
		  due_at,
		  last_grade,
		  recall_due_at,
		  recall_last_grade,
		  pronunciation_due_at,
		  pronunciation_last_grade
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		itemID, itemType, "unknown", clock.FormatISO(now), clock.FormatISO(now), 1,
		valueOrNil(recallDueAt), valueOrNil(recallGrade), valueOrNil(recallDueAt), valueOrNil(recallGrade),
		valueOrNil(pronunciationDueAt), valueOrNil(pronunciationGrade),
	)
	return err
}

func PreviewDue(days int, now time.Time, reviewType string) (map[string]int, error) {
	expr, err := dueExpr(reviewType)
	if err != nil {
		return nil, err
	}
	dbPath, err := db.InitDB()
	if err != nil {
		return nil, err
	}
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	rows, err := conn.Query(fmt.Sprintf("SELECT %s FROM user_knowledge WHERE %s IS NOT NULL", expr, expr))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	forecast := map[string]int{}
	for rows.Next() {
		var dueAt string
		if err := rows.Scan(&dueAt); err != nil {
			return nil, err
		}
		dueDT, err := clock.ParseUTCISO(dueAt)
		if err != nil {
			return nil, err
		}
		dueDate := time.Date(dueDT.Year(), dueDT.Month(), dueDT.Day(), 0, 0, 0, 0, time.UTC)
		nowDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		delta := int(dueDate.Sub(nowDate).Hours() / 24)
		if delta < 0 || delta > days {
			continue
		}
		key := dueDT.Format("2006-01-02")
		forecast[key] = forecast[key] + 1
	}
	return forecast, rows.Err()
}

func valueOrNil[T any](value *T) any {
	if value == nil {
		return nil
	}
	return *value
}
