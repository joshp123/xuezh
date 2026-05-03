package cram

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

func insertReviewSession(tx *sql.Tx, session storedReviewSession, nowText string) error {
	queueJSON, _ := json.Marshal(session.QueueIDs)
	retryJSON, _ := json.Marshal(session.RetryQueueIDs)
	reviewedJSON, _ := json.Marshal(session.Reviewed)
	repeatJSON, _ := json.Marshal(session.RepeatItemIDs)
	_, err := tx.Exec(`
			INSERT INTO cram_review_sessions (
			  id, status, current_item_id, queue_json, retry_queue_json, reviewed_json,
			  repeat_item_ids_json, revealed, round, was_retry, shown_at, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		session.ID, session.Status, nullText(session.CurrentItemID), string(queueJSON), string(retryJSON), string(reviewedJSON),
		string(repeatJSON), boolInt(session.Revealed), session.Round, boolInt(session.WasRetry), stringPtrValue(session.ShownAt),
		nowText, nowText,
	)
	return err
}

func saveReviewSessionTx(tx *sql.Tx, session *storedReviewSession, nowText string) error {
	queueJSON, _ := json.Marshal(session.QueueIDs)
	retryJSON, _ := json.Marshal(session.RetryQueueIDs)
	reviewedJSON, _ := json.Marshal(session.Reviewed)
	repeatJSON, _ := json.Marshal(session.RepeatItemIDs)
	_, err := tx.Exec(`
			UPDATE cram_review_sessions
			SET status = ?, current_item_id = ?, queue_json = ?, retry_queue_json = ?,
			    reviewed_json = ?, repeat_item_ids_json = ?, revealed = ?, round = ?,
			    was_retry = ?, shown_at = ?, updated_at = ?
			WHERE id = ?`,
		session.Status, nullText(session.CurrentItemID), string(queueJSON), string(retryJSON),
		string(reviewedJSON), string(repeatJSON), boolInt(session.Revealed), session.Round,
		boolInt(session.WasRetry), stringPtrValue(session.ShownAt), nowText, session.ID,
	)
	session.UpdatedAt = nowText
	return err
}

func loadActiveReviewSession(conn *sql.DB) (storedReviewSession, error) {
	return scanStoredReviewSession(conn.QueryRow(`
			SELECT id, status, current_item_id, queue_json, retry_queue_json, reviewed_json,
			       repeat_item_ids_json, revealed, round, was_retry, shown_at, updated_at
			FROM cram_review_sessions
			WHERE status = ?
			ORDER BY updated_at DESC
			LIMIT 1`, reviewSessionActive))
}

func loadReviewSessionTx(tx *sql.Tx, id string) (storedReviewSession, error) {
	if id == "" {
		return storedReviewSession{}, fmt.Errorf("session_id is required")
	}
	return scanStoredReviewSession(tx.QueryRow(`
			SELECT id, status, current_item_id, queue_json, retry_queue_json, reviewed_json,
			       repeat_item_ids_json, revealed, round, was_retry, shown_at, updated_at
			FROM cram_review_sessions
			WHERE id = ?`, id))
}

func scanStoredReviewSession(row *sql.Row) (storedReviewSession, error) {
	var session storedReviewSession
	var current, shownAt sql.NullString
	var queueJSON, retryJSON, reviewedJSON, repeatJSON string
	var revealed, wasRetry int
	if err := row.Scan(
		&session.ID, &session.Status, &current, &queueJSON, &retryJSON, &reviewedJSON,
		&repeatJSON, &revealed, &session.Round, &wasRetry, &shownAt, &session.UpdatedAt,
	); err != nil {
		return storedReviewSession{}, err
	}
	session.CurrentItemID = stringFromNull(current)
	session.Revealed = revealed != 0
	session.WasRetry = wasRetry != 0
	session.ShownAt = nullStringPtr(shownAt)
	_ = json.Unmarshal([]byte(queueJSON), &session.QueueIDs)
	_ = json.Unmarshal([]byte(retryJSON), &session.RetryQueueIDs)
	_ = json.Unmarshal([]byte(reviewedJSON), &session.Reviewed)
	_ = json.Unmarshal([]byte(repeatJSON), &session.RepeatItemIDs)
	if session.Round <= 0 {
		session.Round = 1
	}
	return session, nil
}

func reviewSessionState(conn *sql.DB, session storedReviewSession) (ReviewSessionState, error) {
	cardIDs := []string{}
	if session.CurrentItemID != "" {
		cardIDs = append(cardIDs, session.CurrentItemID)
	}
	cardIDs = append(cardIDs, session.QueueIDs...)
	cardIDs = append(cardIDs, session.RetryQueueIDs...)
	for _, reviewed := range session.Reviewed {
		cardIDs = append(cardIDs, reviewed.ItemID)
	}
	cards, err := cardsByID(conn, cardIDs)
	if err != nil {
		return ReviewSessionState{}, err
	}
	var current *Card
	if session.CurrentItemID != "" {
		if card, ok := cards[session.CurrentItemID]; ok {
			copy := card
			current = &copy
		}
	}
	state := ReviewSessionState{
		ID:            session.ID,
		Status:        session.Status,
		Card:          current,
		Queue:         cardsInOrder(cards, session.QueueIDs),
		RetryQueue:    cardsInOrder(cards, session.RetryQueueIDs),
		RepeatItemIDs: append([]string{}, session.RepeatItemIDs...),
		Revealed:      session.Revealed,
		Round:         session.Round,
		WasRetry:      session.WasRetry,
		ShownAt:       copyStringPtr(session.ShownAt),
		UpdatedAt:     session.UpdatedAt,
	}
	for _, reviewed := range session.Reviewed {
		card, ok := cards[reviewed.ItemID]
		if !ok {
			continue
		}
		state.ReviewedCards = append(state.ReviewedCards, ReviewedSessionCard{
			Card:   card,
			Grade:  reviewed.Grade,
			Repeat: reviewed.Repeat,
		})
	}
	return state, nil
}

func cardsByID(conn *sql.DB, ids []string) (map[string]Card, error) {
	unique := []string{}
	seen := map[string]bool{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" && !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	if len(unique) == 0 {
		return map[string]Card{}, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(unique)), ",")
	args := make([]any, 0, len(unique))
	for _, id := range unique {
		args = append(args, id)
	}
	rows, err := conn.Query(`
			SELECT id, source, category, learning_order, source_index, pinyin, hanzi, meaning,
			       sentence_pinyin, sentence_hanzi, sentence_hanzi_raw, sentence_meaning,
			       row_hash, sentence_audio_paths_json
			FROM cram_items
			WHERE id IN (`+placeholders+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cards := map[string]Card{}
	for rows.Next() {
		item, err := scanItemRow(rows)
		if err != nil {
			return nil, err
		}
		cards[item.ID] = cardFromItem(item)
	}
	return cards, rows.Err()
}

func cardsInOrder(cards map[string]Card, ids []string) []Card {
	ordered := []Card{}
	for _, id := range ids {
		if card, ok := cards[id]; ok {
			ordered = append(ordered, card)
		}
	}
	return ordered
}

func copyStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func nullText(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func containsID(ids []string, id string) bool {
	for _, existing := range ids {
		if existing == id {
			return true
		}
	}
	return false
}

func toggleID(ids []string, id string) []string {
	if containsID(ids, id) {
		return removeID(ids, id)
	}
	return append(ids, id)
}

func removeID(ids []string, id string) []string {
	next := ids[:0]
	for _, existing := range ids {
		if existing != id {
			next = append(next, existing)
		}
	}
	return next
}
