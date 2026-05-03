package cram

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/joshp123/xuezh/internal/xuezh/clock"
)

const (
	reviewSessionActive = "active"
	reviewSessionDone   = "done"
)

type storedReviewSession struct {
	ID            string
	Status        string
	CurrentItemID string
	QueueIDs      []string
	RetryQueueIDs []string
	Reviewed      []storedReviewedCard
	RepeatItemIDs []string
	Revealed      bool
	Round         int
	WasRetry      bool
	ShownAt       *string
	UpdatedAt     string
}

type storedReviewedCard struct {
	ItemID   string                `json:"item_id"`
	Grade    string                `json:"grade"`
	Repeat   bool                  `json:"repeat"`
	Snapshot storedSessionSnapshot `json:"snapshot"`
}

type storedSessionSnapshot struct {
	CurrentItemID string   `json:"current_item_id"`
	QueueIDs      []string `json:"queue_ids"`
	RetryQueueIDs []string `json:"retry_queue_ids"`
	RepeatItemIDs []string `json:"repeat_item_ids"`
	Revealed      bool     `json:"revealed"`
	Round         int      `json:"round"`
	WasRetry      bool     `json:"was_retry"`
	ShownAt       *string  `json:"shown_at"`
}

func ActiveReviewSession() (*ReviewSessionState, error) {
	conn, err := openDB()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	stored, err := loadActiveReviewSession(conn)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	state, err := reviewSessionState(conn, stored)
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func StartReviewSession(opts ReviewSessionStartOptions, now time.Time) (ReviewSessionState, error) {
	conn, err := openDB()
	if err != nil {
		return ReviewSessionState{}, err
	}
	defer conn.Close()
	cards, err := NextCards(NextOptions{Limit: opts.Limit, CardIDs: opts.CardIDs}, now)
	if err != nil {
		return ReviewSessionState{}, err
	}
	tx, err := conn.Begin()
	if err != nil {
		return ReviewSessionState{}, err
	}
	defer tx.Rollback()
	nowText := clock.FormatISO(now)
	if _, err := tx.Exec("UPDATE cram_review_sessions SET status = ?, updated_at = ? WHERE status = ?", reviewSessionDone, nowText, reviewSessionActive); err != nil {
		return ReviewSessionState{}, err
	}
	session := storedReviewSession{
		ID:        newID(),
		Status:    reviewSessionActive,
		QueueIDs:  []string{},
		Round:     1,
		UpdatedAt: nowText,
	}
	if len(cards) > 0 {
		session.CurrentItemID = cards[0].ItemID
		session.ShownAt = &nowText
		for _, card := range cards[1:] {
			session.QueueIDs = append(session.QueueIDs, card.ItemID)
		}
	} else {
		session.Status = reviewSessionDone
	}
	if err := insertReviewSession(tx, session, nowText); err != nil {
		return ReviewSessionState{}, err
	}
	if err := tx.Commit(); err != nil {
		return ReviewSessionState{}, err
	}
	return reviewSessionState(conn, session)
}

func RevealReviewSession(sessionID string, now time.Time) (ReviewSessionState, error) {
	return updateReviewSession(sessionID, now, func(session *storedReviewSession) error {
		session.Revealed = true
		return nil
	})
}

func ToggleReviewSessionRepeat(sessionID string, now time.Time) (ReviewSessionState, error) {
	return updateReviewSession(sessionID, now, func(session *storedReviewSession) error {
		if session.CurrentItemID == "" {
			return nil
		}
		session.RepeatItemIDs = toggleID(session.RepeatItemIDs, session.CurrentItemID)
		return nil
	})
}

func GradeReviewSession(opts GradeOptions, now time.Time) (ReviewSessionState, GradeResult, error) {
	sessionID := strings.TrimSpace(opts.SessionID)
	if sessionID == "" {
		return ReviewSessionState{}, GradeResult{}, fmt.Errorf("session_id is required")
	}
	conn, err := openDB()
	if err != nil {
		return ReviewSessionState{}, GradeResult{}, err
	}
	defer conn.Close()
	tx, err := conn.Begin()
	if err != nil {
		return ReviewSessionState{}, GradeResult{}, err
	}
	defer tx.Rollback()

	session, err := loadReviewSessionTx(tx, sessionID)
	if err != nil {
		return ReviewSessionState{}, GradeResult{}, err
	}
	if session.Status != reviewSessionActive || session.CurrentItemID == "" {
		return ReviewSessionState{}, GradeResult{}, fmt.Errorf("no active card in session")
	}
	if strings.TrimSpace(opts.ItemID) != session.CurrentItemID {
		return ReviewSessionState{}, GradeResult{}, fmt.Errorf("answer is for a different card")
	}
	if !session.Revealed {
		return ReviewSessionState{}, GradeResult{}, fmt.Errorf("card is not revealed")
	}
	opts.Round = session.Round
	opts.WasRetry = session.WasRetry
	opts.ShownAt = stringPtrText(session.ShownAt)
	result, err := gradeCardTx(tx, opts, now)
	if err != nil {
		return ReviewSessionState{}, GradeResult{}, err
	}
	repeat := containsID(session.RepeatItemIDs, session.CurrentItemID)
	snapshot := session.snapshot()
	session.Reviewed = append([]storedReviewedCard{{
		ItemID:   session.CurrentItemID,
		Grade:    opts.Grade,
		Repeat:   repeat,
		Snapshot: snapshot,
	}}, session.Reviewed...)
	session.RepeatItemIDs = removeID(session.RepeatItemIDs, session.CurrentItemID)
	extraRetry := []string{}
	if opts.Grade == GradeIncorrect || repeat {
		extraRetry = append(extraRetry, session.CurrentItemID)
	}
	session.advance(extraRetry, now)
	if err := saveReviewSessionTx(tx, &session, clock.FormatISO(now)); err != nil {
		return ReviewSessionState{}, GradeResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return ReviewSessionState{}, GradeResult{}, err
	}
	state, err := reviewSessionState(conn, session)
	if err != nil {
		return ReviewSessionState{}, GradeResult{}, err
	}
	return state, result, nil
}

func UndoReviewSession(sessionID string, now time.Time) (ReviewSessionState, UndoGradeResult, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return ReviewSessionState{}, UndoGradeResult{}, fmt.Errorf("session_id is required")
	}
	conn, err := openDB()
	if err != nil {
		return ReviewSessionState{}, UndoGradeResult{}, err
	}
	defer conn.Close()
	tx, err := conn.Begin()
	if err != nil {
		return ReviewSessionState{}, UndoGradeResult{}, err
	}
	defer tx.Rollback()

	session, err := loadReviewSessionTx(tx, sessionID)
	if err != nil {
		return ReviewSessionState{}, UndoGradeResult{}, err
	}
	if len(session.Reviewed) == 0 {
		return ReviewSessionState{}, UndoGradeResult{}, fmt.Errorf("no grade to undo")
	}
	last := session.Reviewed[0]
	result, err := undoLastGradeTx(tx, UndoGradeOptions{SessionID: sessionID, ItemID: last.ItemID}, now)
	if err != nil {
		return ReviewSessionState{}, UndoGradeResult{}, err
	}
	session.restore(last.Snapshot)
	session.Reviewed = session.Reviewed[1:]
	session.Status = reviewSessionActive
	if err := saveReviewSessionTx(tx, &session, clock.FormatISO(now)); err != nil {
		return ReviewSessionState{}, UndoGradeResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return ReviewSessionState{}, UndoGradeResult{}, err
	}
	settings, err := loadLocalScoringSettings(conn)
	if err != nil {
		return ReviewSessionState{}, UndoGradeResult{}, err
	}
	state, err := reviewSessionState(conn, session)
	if err != nil {
		return ReviewSessionState{}, UndoGradeResult{}, err
	}
	return state, finishUndoGradeResult(result, settings), nil
}

func updateReviewSession(sessionID string, now time.Time, update func(*storedReviewSession) error) (ReviewSessionState, error) {
	conn, err := openDB()
	if err != nil {
		return ReviewSessionState{}, err
	}
	defer conn.Close()
	tx, err := conn.Begin()
	if err != nil {
		return ReviewSessionState{}, err
	}
	defer tx.Rollback()
	session, err := loadReviewSessionTx(tx, strings.TrimSpace(sessionID))
	if err != nil {
		return ReviewSessionState{}, err
	}
	if err := update(&session); err != nil {
		return ReviewSessionState{}, err
	}
	if err := saveReviewSessionTx(tx, &session, clock.FormatISO(now)); err != nil {
		return ReviewSessionState{}, err
	}
	if err := tx.Commit(); err != nil {
		return ReviewSessionState{}, err
	}
	return reviewSessionState(conn, session)
}

func (session storedReviewSession) snapshot() storedSessionSnapshot {
	return storedSessionSnapshot{
		CurrentItemID: session.CurrentItemID,
		QueueIDs:      append([]string{}, session.QueueIDs...),
		RetryQueueIDs: append([]string{}, session.RetryQueueIDs...),
		RepeatItemIDs: append([]string{}, session.RepeatItemIDs...),
		Revealed:      session.Revealed,
		Round:         session.Round,
		WasRetry:      session.WasRetry,
		ShownAt:       copyStringPtr(session.ShownAt),
	}
}

func (session *storedReviewSession) restore(snapshot storedSessionSnapshot) {
	session.CurrentItemID = snapshot.CurrentItemID
	session.QueueIDs = append([]string{}, snapshot.QueueIDs...)
	session.RetryQueueIDs = append([]string{}, snapshot.RetryQueueIDs...)
	session.RepeatItemIDs = append([]string{}, snapshot.RepeatItemIDs...)
	session.Revealed = snapshot.Revealed
	session.Round = snapshot.Round
	session.WasRetry = snapshot.WasRetry
	session.ShownAt = copyStringPtr(snapshot.ShownAt)
}

func (session *storedReviewSession) advance(extraRetry []string, now time.Time) {
	session.RetryQueueIDs = append(session.RetryQueueIDs, extraRetry...)
	session.Revealed = false
	nowText := clock.FormatISO(now)
	session.ShownAt = &nowText
	if len(session.QueueIDs) > 0 {
		session.CurrentItemID = session.QueueIDs[0]
		session.QueueIDs = session.QueueIDs[1:]
		session.WasRetry = false
		return
	}
	if len(session.RetryQueueIDs) > 0 {
		session.CurrentItemID = session.RetryQueueIDs[0]
		session.RetryQueueIDs = session.RetryQueueIDs[1:]
		session.WasRetry = true
		session.Round++
		return
	}
	session.CurrentItemID = ""
	session.ShownAt = nil
	session.Status = reviewSessionDone
}
