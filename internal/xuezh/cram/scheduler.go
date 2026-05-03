package cram

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/joshp123/xuezh/internal/xuezh/clock"
)

type stateSnapshot struct {
	Source    string
	Category  string
	NextDueAt *string
	Score     scoreState
}

type gradePayload struct {
	Mode       string         `json:"mode"`
	Source     string         `json:"source"`
	Category   string         `json:"category"`
	Grade      string         `json:"grade"`
	OldDueAt   *string        `json:"old_due_at"`
	OldScore   scoreStateJSON `json:"old_score"`
	NewScore   scoreStateJSON `json:"new_score"`
	Scored     bool           `json:"scored"`
	WasRetry   bool           `json:"was_retry"`
	Round      int            `json:"round"`
	ElapsedMS  int            `json:"elapsed_ms"`
	ShownAt    string         `json:"shown_at,omitempty"`
	AnsweredAt string         `json:"answered_at,omitempty"`
}

type scoreStateJSON struct {
	CardID          int     `json:"card_id"`
	Score           *int    `json:"score"`
	Difficulty      int     `json:"difficulty"`
	History         string  `json:"history"`
	Correct         int     `json:"correct"`
	Incorrect       int     `json:"incorrect"`
	Reviewed        int     `json:"reviewed"`
	FirstReviewedAt *string `json:"first_reviewed_at"`
	LastReviewedAt  *string `json:"last_reviewed_at"`
	SinceLastChange int     `json:"sincelastchange"`
	ScoreIncTime    int64   `json:"scoreinctime"`
	ScoreDecTime    int64   `json:"scoredectime"`
}

func GradeCard(opts GradeOptions, now time.Time) (GradeResult, error) {
	itemID := strings.TrimSpace(opts.ItemID)
	if itemID == "" {
		return GradeResult{}, fmt.Errorf("item is required")
	}
	grade := strings.TrimSpace(opts.Grade)
	if grade != GradeIncorrect && grade != GradeCorrect {
		return GradeResult{}, fmt.Errorf("invalid grade: %s", grade)
	}
	conn, err := openDB()
	if err != nil {
		return GradeResult{}, err
	}
	defer conn.Close()

	tx, err := conn.Begin()
	if err != nil {
		return GradeResult{}, err
	}
	defer tx.Rollback()

	result, err := gradeCardTx(tx, opts, now)
	if err != nil {
		return GradeResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return GradeResult{}, err
	}
	return result, nil
}

func gradeCardTx(tx *sql.Tx, opts GradeOptions, now time.Time) (GradeResult, error) {
	itemID := strings.TrimSpace(opts.ItemID)
	if itemID == "" {
		return GradeResult{}, fmt.Errorf("item is required")
	}
	grade := strings.TrimSpace(opts.Grade)
	if grade != GradeIncorrect && grade != GradeCorrect {
		return GradeResult{}, fmt.Errorf("invalid grade: %s", grade)
	}
	before, err := loadStateSnapshot(tx, itemID)
	if err == sql.ErrNoRows {
		return GradeResult{}, fmt.Errorf("item not found: %s", itemID)
	}
	if err != nil {
		return GradeResult{}, err
	}
	settings, err := loadLocalScoringSettings(tx)
	if err != nil {
		return GradeResult{}, err
	}

	update := applyPlecoAnswer(before.Score, settings, grade, now)
	after := update.After
	score := scoreValue(after.Score)
	interval := 0
	nextDue := before.NextDueAt
	if update.Scored {
		interval = intervalMinutesForScore(score, settings)
		value := clock.FormatISO(now.Add(time.Duration(interval) * time.Minute))
		nextDue = &value
		if err := saveScoreState(tx, itemID, nextDue, after, clock.FormatISO(now)); err != nil {
			return GradeResult{}, err
		}
	} else if after.Score != nil {
		interval = intervalMinutesForScore(score, settings)
	}

	sessionID := strings.TrimSpace(opts.SessionID)
	if sessionID == "" {
		sessionID = newID()
	}
	payload := gradePayload{
		Mode:       "cram",
		Source:     before.Source,
		Category:   before.Category,
		Grade:      grade,
		OldDueAt:   before.NextDueAt,
		OldScore:   scoreStateToJSON(before.Score),
		NewScore:   scoreStateToJSON(after),
		Scored:     update.Scored,
		WasRetry:   opts.WasRetry,
		Round:      opts.Round,
		ElapsedMS:  opts.ElapsedMS,
		ShownAt:    opts.ShownAt,
		AnsweredAt: opts.AnsweredAt,
	}
	payloadJSON, _ := json.Marshal(payload)
	nowText := clock.FormatISO(now)
	if _, err := tx.Exec(
		"INSERT INTO review_events (id, item_id, event_type, ts, session_id, payload_json) VALUES (?, ?, ?, ?, ?, ?)",
		newID(), itemID, "cram.grade", nowText, sessionID, string(payloadJSON),
	); err != nil {
		return GradeResult{}, err
	}
	return GradeResult{
		ItemID:          itemID,
		Grade:           grade,
		NextDueAt:       stringPtrText(nextDue),
		IntervalMinutes: interval,
		Score:           score,
		Difficulty:      after.Difficulty,
		CorrectCount:    after.Correct,
		IncorrectCount:  after.Incorrect,
		ReviewedCount:   after.Reviewed,
		Scored:          update.Scored,
	}, nil
}

func UndoLastGrade(opts UndoGradeOptions) (UndoGradeResult, error) {
	sessionID := strings.TrimSpace(opts.SessionID)
	if sessionID == "" {
		return UndoGradeResult{}, fmt.Errorf("session_id is required")
	}
	itemID := strings.TrimSpace(opts.ItemID)
	if itemID == "" {
		return UndoGradeResult{}, fmt.Errorf("item is required")
	}
	conn, err := openDB()
	if err != nil {
		return UndoGradeResult{}, err
	}
	defer conn.Close()

	tx, err := conn.Begin()
	if err != nil {
		return UndoGradeResult{}, err
	}
	defer tx.Rollback()

	result, err := undoLastGradeTx(tx, opts, time.Now().UTC())
	if err != nil {
		return UndoGradeResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return UndoGradeResult{}, err
	}
	settings, err := loadLocalScoringSettings(conn)
	if err != nil {
		return UndoGradeResult{}, err
	}
	return finishUndoGradeResult(result, settings), nil
}

func undoLastGradeTx(tx *sql.Tx, opts UndoGradeOptions, now time.Time) (UndoGradeResult, error) {
	sessionID := strings.TrimSpace(opts.SessionID)
	if sessionID == "" {
		return UndoGradeResult{}, fmt.Errorf("session_id is required")
	}
	itemID := strings.TrimSpace(opts.ItemID)
	if itemID == "" {
		return UndoGradeResult{}, fmt.Errorf("item is required")
	}
	var eventID, eventItemID, payloadText string
	err := tx.QueryRow(`
		SELECT id, item_id, payload_json
		FROM review_events
		WHERE event_type = 'cram.grade' AND session_id = ?
		ORDER BY ts DESC, rowid DESC
		LIMIT 1`, sessionID).Scan(&eventID, &eventItemID, &payloadText)
	if err == sql.ErrNoRows {
		return UndoGradeResult{}, fmt.Errorf("no grade to undo")
	}
	if err != nil {
		return UndoGradeResult{}, err
	}
	if eventItemID != itemID {
		return UndoGradeResult{}, fmt.Errorf("latest grade is for a different card")
	}

	var payload gradePayload
	if err := json.Unmarshal([]byte(payloadText), &payload); err != nil {
		return UndoGradeResult{}, err
	}
	oldScore := scoreStateFromJSON(payload.OldScore)
	if err := saveScoreState(tx, itemID, payload.OldDueAt, oldScore, clock.FormatISO(now)); err != nil {
		return UndoGradeResult{}, err
	}
	if _, err := tx.Exec("DELETE FROM review_events WHERE id = ?", eventID); err != nil {
		return UndoGradeResult{}, err
	}
	return UndoGradeResult{
		ItemID:         itemID,
		UndoneGrade:    payload.Grade,
		NextDueAt:      payload.OldDueAt,
		Score:          oldScore.Score,
		Difficulty:     oldScore.Difficulty,
		CorrectCount:   oldScore.Correct,
		IncorrectCount: oldScore.Incorrect,
		ReviewedCount:  oldScore.Reviewed,
	}, nil
}

func finishUndoGradeResult(result UndoGradeResult, settings scoringSettings) UndoGradeResult {
	if result.Score != nil {
		result.IntervalMinutes = intervalMinutesForScore(*result.Score, settings)
	}
	return result
}

func loadStateSnapshot(tx *sql.Tx, itemID string) (stateSnapshot, error) {
	var row stateSnapshot
	var nextDue, history, firstReviewed, lastReviewed sql.NullString
	var cardID, score, difficulty, correct, incorrect, reviewed, sinceLastChange, scoreIncTime, scoreDecTime sql.NullInt64
	err := tx.QueryRow(`
		SELECT i.source, i.category, s.next_due_at,
		       s.pleco_card_id, s.pleco_score, s.pleco_difficulty, s.pleco_history,
		       s.pleco_correct_count, s.pleco_incorrect_count, s.pleco_reviewed_count,
		       s.pleco_first_reviewed_at, s.pleco_last_reviewed_at,
		       s.pleco_sincelastchange, s.pleco_score_inc_time, s.pleco_score_dec_time
		FROM cram_items i
		JOIN cram_state s ON s.item_id = i.id
		WHERE i.id = ?`, itemID).
		Scan(
			&row.Source, &row.Category, &nextDue,
			&cardID, &score, &difficulty, &history, &correct, &incorrect,
			&reviewed, &firstReviewed, &lastReviewed, &sinceLastChange,
			&scoreIncTime, &scoreDecTime,
		)
	if err != nil {
		return stateSnapshot{}, err
	}
	row.NextDueAt = nullStringPtr(nextDue)
	row.Score = scoreState{
		CardID:          intFromNull(cardID),
		Score:           intPtrFromNull(score),
		Difficulty:      intFromNull(difficulty),
		History:         stringFromNull(history),
		Correct:         intFromNull(correct),
		Incorrect:       intFromNull(incorrect),
		Reviewed:        intFromNull(reviewed),
		FirstReviewedAt: timePtrFromNull(firstReviewed),
		LastReviewedAt:  timePtrFromNull(lastReviewed),
		SinceLastChange: intFromNull(sinceLastChange),
		ScoreIncTime:    int64FromNull(scoreIncTime),
		ScoreDecTime:    int64FromNull(scoreDecTime),
	}
	return row, nil
}

func saveScoreState(tx *sql.Tx, itemID string, nextDue *string, score scoreState, nowText string) error {
	_, err := tx.Exec(`
		UPDATE cram_state
		SET next_due_at = ?,
		    pleco_score = ?, pleco_difficulty = ?, pleco_history = ?,
		    pleco_correct_count = ?, pleco_incorrect_count = ?, pleco_reviewed_count = ?,
		    pleco_first_reviewed_at = ?, pleco_last_reviewed_at = ?,
		    pleco_sincelastchange = ?, pleco_score_inc_time = ?, pleco_score_dec_time = ?,
		    updated_at = ?
		WHERE item_id = ?`,
		stringPtrValue(nextDue),
		intPtrValue(score.Score), nullableScoreInt(score.Score, score.Difficulty), nullableScoreString(score.Score, score.History),
		nullableScoreInt(score.Score, score.Correct), nullableScoreInt(score.Score, score.Incorrect), nullableScoreInt(score.Score, score.Reviewed),
		timePtrValue(score.FirstReviewedAt), timePtrValue(score.LastReviewedAt),
		nullableScoreInt(score.Score, score.SinceLastChange), nullableScoreInt64(score.Score, score.ScoreIncTime), nullableScoreInt64(score.Score, score.ScoreDecTime),
		nowText, itemID)
	return err
}

func scoreStateToJSON(score scoreState) scoreStateJSON {
	return scoreStateJSON{
		CardID:          score.CardID,
		Score:           score.Score,
		Difficulty:      score.Difficulty,
		History:         score.History,
		Correct:         score.Correct,
		Incorrect:       score.Incorrect,
		Reviewed:        score.Reviewed,
		FirstReviewedAt: timePtrValue(score.FirstReviewedAt),
		LastReviewedAt:  timePtrValue(score.LastReviewedAt),
		SinceLastChange: score.SinceLastChange,
		ScoreIncTime:    score.ScoreIncTime,
		ScoreDecTime:    score.ScoreDecTime,
	}
}

func scoreStateFromJSON(payload scoreStateJSON) scoreState {
	return scoreState{
		CardID:          payload.CardID,
		Score:           payload.Score,
		Difficulty:      payload.Difficulty,
		History:         payload.History,
		Correct:         payload.Correct,
		Incorrect:       payload.Incorrect,
		Reviewed:        payload.Reviewed,
		FirstReviewedAt: timePtrFromString(payload.FirstReviewedAt),
		LastReviewedAt:  timePtrFromString(payload.LastReviewedAt),
		SinceLastChange: payload.SinceLastChange,
		ScoreIncTime:    payload.ScoreIncTime,
		ScoreDecTime:    payload.ScoreDecTime,
	}
}

func nullStringPtr(value sql.NullString) *string {
	if value.Valid {
		return &value.String
	}
	return nil
}

func intPtrFromNull(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	intValue := int(value.Int64)
	return &intValue
}

func intFromNull(value sql.NullInt64) int {
	if value.Valid {
		return int(value.Int64)
	}
	return 0
}

func int64FromNull(value sql.NullInt64) int64 {
	if value.Valid {
		return value.Int64
	}
	return 0
}

func stringFromNull(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}

func timePtrFromNull(value sql.NullString) *time.Time {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil
	}
	return timePtrFromString(&value.String)
}

func timePtrFromString(value *string) *time.Time {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, *value)
	if err != nil {
		return nil
	}
	return &parsed
}

func intPtrValue(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func stringPtrValue(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func stringPtrText(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func timePtrValue(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := clock.FormatISO(*value)
	return &formatted
}

func nullableScoreInt(score *int, value int) any {
	if score == nil {
		return nil
	}
	return value
}

func nullableScoreInt64(score *int, value int64) any {
	if score == nil {
		return nil
	}
	return value
}

func nullableScoreString(score *int, value string) any {
	if score == nil {
		return nil
	}
	return value
}

func scoreValue(score *int) int {
	if score == nil {
		return 0
	}
	return *score
}
