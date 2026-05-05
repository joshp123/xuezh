package cram

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/joshp123/xuezh/internal/xuezh/clock"
)

var learnerStateColumns = []string{
	"category",
	"hanzi",
	"meaning",
	"sentence",
	"sentence_meaning",
	"score",
	"learned",
	"due",
	"correct",
	"incorrect",
	"reviewed",
	"first_reviewed",
	"last_reviewed",
	"next_due",
	"history",
}

func LearnerStateFor(now time.Time) (LearnerState, error) {
	conn, err := openDB()
	if err != nil {
		return LearnerState{}, err
	}
	defer conn.Close()

	settings, err := loadLocalScoringSettings(conn)
	if err != nil {
		return LearnerState{}, err
	}
	changedAt, err := learnerChangedAt(conn, now)
	if err != nil {
		return LearnerState{}, err
	}

	rows, err := conn.Query(`
		SELECT i.source, i.category, i.hanzi, i.meaning, i.sentence_hanzi, i.sentence_meaning,
		       s.pleco_score, s.pleco_correct_count, s.pleco_incorrect_count,
		       s.pleco_reviewed_count, s.pleco_first_reviewed_at, s.pleco_last_reviewed_at,
		       s.next_due_at, s.pleco_history
		FROM cram_items i
		JOIN cram_state s ON s.item_id = i.id
		ORDER BY i.source, i.learning_order`)
	if err != nil {
		return LearnerState{}, err
	}
	defer rows.Close()

	state := LearnerState{
		GeneratedAt:  clock.FormatISO(now),
		ChangedAt:    changedAt,
		LearnedScore: settings.LearnedScore,
		Columns:      append([]string{}, learnerStateColumns...),
		Cards:        [][]any{},
	}
	for rows.Next() {
		var source string
		var category string
		var hanzi string
		var meaning string
		var sentence string
		var sentenceMeaning sql.NullString
		var score sql.NullInt64
		var correct sql.NullInt64
		var incorrect sql.NullInt64
		var reviewed sql.NullInt64
		var firstReviewed sql.NullString
		var lastReviewed sql.NullString
		var nextDue sql.NullString
		var history sql.NullString
		if err := rows.Scan(
			&source, &category, &hanzi, &meaning, &sentence, &sentenceMeaning,
			&score, &correct, &incorrect, &reviewed, &firstReviewed, &lastReviewed,
			&nextDue, &history,
		); err != nil {
			return LearnerState{}, err
		}
		state.Cards = append(state.Cards, []any{
			learnerCategory(source, category),
			hanzi,
			meaning,
			sentence,
			nullString(sentenceMeaning),
			nullInt(score),
			score.Valid && int(score.Int64) >= settings.LearnedScore,
			isDue(nextDue, now),
			intOrZero(correct),
			intOrZero(incorrect),
			intOrZero(reviewed),
			nullString(firstReviewed),
			nullString(lastReviewed),
			nullString(nextDue),
			nullString(history),
		})
	}
	if err := rows.Err(); err != nil {
		return LearnerState{}, err
	}
	hash, err := learnerStateHash(state)
	if err != nil {
		return LearnerState{}, err
	}
	state.StateHash = hash
	return state, nil
}

func learnerCategory(source, category string) string {
	label := sourceLabel(strings.TrimSpace(source))
	name := strings.TrimSpace(category)
	if label == "" {
		return name
	}
	if name == "" {
		return label
	}
	return label + " / " + name
}

func learnerChangedAt(conn *sql.DB, now time.Time) (string, error) {
	var changed sql.NullString
	err := conn.QueryRow(`
		SELECT MAX(updated_at) FROM (
			SELECT updated_at FROM cram_items
			UNION ALL
			SELECT updated_at FROM cram_state
		)`).Scan(&changed)
	if err != nil {
		return "", err
	}
	if changed.Valid && strings.TrimSpace(changed.String) != "" {
		return changed.String, nil
	}
	return clock.FormatISO(now), nil
}

func learnerStateHash(state LearnerState) (string, error) {
	payload := struct {
		ChangedAt    string   `json:"changed_at"`
		LearnedScore int      `json:"learned_score"`
		Columns      []string `json:"columns"`
		Cards        [][]any  `json:"cards"`
	}{
		ChangedAt:    state.ChangedAt,
		LearnedScore: state.LearnedScore,
		Columns:      state.Columns,
		Cards:        state.Cards,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func nullString(value sql.NullString) any {
	if value.Valid && strings.TrimSpace(value.String) != "" {
		return value.String
	}
	return nil
}

func nullInt(value sql.NullInt64) any {
	if value.Valid {
		return int(value.Int64)
	}
	return nil
}

func intOrZero(value sql.NullInt64) int {
	if value.Valid {
		return int(value.Int64)
	}
	return 0
}

func isDue(nextDue sql.NullString, now time.Time) bool {
	if !nextDue.Valid || strings.TrimSpace(nextDue.String) == "" {
		return false
	}
	parsed, err := time.Parse(time.RFC3339, nextDue.String)
	if err != nil {
		return false
	}
	return !parsed.After(now)
}
