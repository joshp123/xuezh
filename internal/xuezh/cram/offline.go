package cram

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/joshp123/xuezh/internal/xuezh/clock"
)

func OfflineDeck(now time.Time) (OfflineDeckSnapshot, error) {
	conn, err := openDB()
	if err != nil {
		return OfflineDeckSnapshot{}, err
	}
	defer conn.Close()
	settings, err := loadLocalScoringSettings(conn)
	if err != nil {
		return OfflineDeckSnapshot{}, err
	}
	rows, err := conn.Query(`
		SELECT i.id, i.source, i.category, i.learning_order, i.source_index, i.pinyin, i.hanzi, i.meaning,
		       i.sentence_pinyin, i.sentence_hanzi, i.sentence_hanzi_raw, i.sentence_meaning,
		       i.row_hash, i.sentence_audio_paths_json, s.next_due_at,
		       s.pleco_score, s.pleco_difficulty, s.pleco_correct_count, s.pleco_incorrect_count,
		       s.pleco_reviewed_count, s.pleco_first_reviewed_at, s.pleco_last_reviewed_at
		FROM cram_items i
		JOIN cram_state s ON s.item_id = i.id
		ORDER BY i.source, i.learning_order`)
	if err != nil {
		return OfflineDeckSnapshot{}, err
	}
	defer rows.Close()

	audioSeen := map[string]bool{}
	audioPaths := []string{}
	cards := []OfflineDeckCard{}
	for rows.Next() {
		var item itemRow
		var sourceIndex sql.NullInt64
		var audioJSON string
		var due, firstReviewed, lastReviewed sql.NullString
		var score, difficulty, correct, incorrect, reviewed sql.NullInt64
		if err := rows.Scan(
			&item.ID, &item.Source, &item.Category, &item.LearningOrder, &sourceIndex,
			&item.Pinyin, &item.Hanzi, &item.Meaning, &item.SentencePinyin,
			&item.SentenceHanzi, &item.SentenceHanziRaw, &item.SentenceMeaning,
			&item.RowHash, &audioJSON, &due, &score, &difficulty, &correct,
			&incorrect, &reviewed, &firstReviewed, &lastReviewed,
		); err != nil {
			return OfflineDeckSnapshot{}, err
		}
		if sourceIndex.Valid {
			item.SourceIndex = int(sourceIndex.Int64)
		}
		item.SentenceAudioPaths = map[string]string{}
		_ = json.Unmarshal([]byte(audioJSON), &item.SentenceAudioPaths)
		card := OfflineDeckCard{
			Card:            cardFromItem(item),
			Score:           intPtrFromNull(score),
			Difficulty:      intFromNull(difficulty),
			CorrectCount:    intFromNull(correct),
			IncorrectCount:  intFromNull(incorrect),
			ReviewedCount:   intFromNull(reviewed),
			FirstReviewedAt: nullStringPtr(firstReviewed),
			LastReviewedAt:  nullStringPtr(lastReviewed),
		}
		card.DueAt = nullStringPtr(due)
		for _, path := range card.SentenceAudioPaths {
			if strings.TrimSpace(path) != "" && !audioSeen[path] {
				audioSeen[path] = true
				audioPaths = append(audioPaths, path)
			}
		}
		cards = append(cards, card)
	}
	if err := rows.Err(); err != nil {
		return OfflineDeckSnapshot{}, err
	}
	sort.Strings(audioPaths)
	return OfflineDeckSnapshot{
		GeneratedAt: clock.FormatISO(now),
		Cards:       cards,
		AudioPaths:  audioPaths,
		Settings: OfflineScoringSettings{
			LearnedScore:              settings.LearnedScore,
			MinScore:                  settings.MinScore,
			MaxScore:                  settings.MaxScore,
			MinDifficulty:             settings.MinDifficulty,
			MaxDifficulty:             settings.MaxDifficulty,
			PointsPerDay:              settings.PointsPerDay,
			ScoreOncePerDay:           settings.ScoreOncePerDay,
			IncorrectScore:            settings.IncorrectScore,
			CorrectInitialScore:       settings.CorrectInitialScore,
			CorrectMultiplier:         settings.CorrectMultiplier,
			CorrectDifficultyChange:   settings.CorrectDifficultyChange,
			IncorrectDifficultyChange: settings.IncorrectDifficultyChange,
			DifficultyDivisor:         settings.DifficultyDivisor,
		},
	}, nil
}

func SyncOfflineReviewEvents(events []OfflineReviewEvent, now time.Time) (OfflineSyncResult, error) {
	events = normalizeOfflineEvents(events, now)
	conn, err := openDB()
	if err != nil {
		return OfflineSyncResult{}, err
	}
	defer conn.Close()
	tx, err := conn.Begin()
	if err != nil {
		return OfflineSyncResult{}, err
	}
	defer tx.Rollback()

	result := OfflineSyncResult{AppliedEventIDs: []string{}, SkippedEventIDs: []string{}}
	for _, event := range events {
		exists, err := reviewEventExists(tx, event.EventID)
		if err != nil {
			return OfflineSyncResult{}, err
		}
		if exists {
			result.Skipped++
			result.SkippedEventIDs = append(result.SkippedEventIDs, event.EventID)
			continue
		}
		answeredAt, err := parseOfflineEventTime(event.AnsweredAt)
		if err != nil {
			return OfflineSyncResult{}, fmt.Errorf("event %s answered_at: %w", event.EventID, err)
		}
		if answeredAt.IsZero() {
			answeredAt = now
			event.AnsweredAt = clock.FormatISO(now)
		}
		if strings.TrimSpace(event.SessionID) == "" {
			event.SessionID = "offline"
		}
		_, err = gradeCardTx(tx, GradeOptions{
			EventID:    event.EventID,
			ItemID:     event.ItemID,
			Grade:      event.Grade,
			SessionID:  event.SessionID,
			ShownAt:    event.ShownAt,
			AnsweredAt: event.AnsweredAt,
			ElapsedMS:  event.ElapsedMS,
			Round:      event.Round,
			WasRetry:   event.WasRetry,
		}, answeredAt)
		if err != nil {
			return OfflineSyncResult{}, fmt.Errorf("event %s: %w", event.EventID, err)
		}
		result.Applied++
		result.AppliedEventIDs = append(result.AppliedEventIDs, event.EventID)
	}
	if err := tx.Commit(); err != nil {
		return OfflineSyncResult{}, err
	}
	return result, nil
}

func normalizeOfflineEvents(events []OfflineReviewEvent, now time.Time) []OfflineReviewEvent {
	normalized := make([]OfflineReviewEvent, 0, len(events))
	for _, event := range events {
		event.EventID = strings.TrimSpace(event.EventID)
		event.ItemID = strings.TrimSpace(event.ItemID)
		event.Grade = strings.TrimSpace(event.Grade)
		event.SessionID = strings.TrimSpace(event.SessionID)
		if event.EventID != "" && event.ItemID != "" {
			normalized = append(normalized, event)
		}
	}
	sort.SliceStable(normalized, func(i, j int) bool {
		left := offlineEventSortTime(normalized[i], now)
		right := offlineEventSortTime(normalized[j], now)
		if !left.Equal(right) {
			return left.Before(right)
		}
		return normalized[i].EventID < normalized[j].EventID
	})
	return normalized
}

func offlineEventSortTime(event OfflineReviewEvent, fallback time.Time) time.Time {
	parsed, err := parseOfflineEventTime(event.AnsweredAt)
	if err != nil || parsed.IsZero() {
		return fallback
	}
	return parsed
}

func parseOfflineEventTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, value)
}

func reviewEventExists(tx *sql.Tx, eventID string) (bool, error) {
	if strings.TrimSpace(eventID) == "" {
		return false, fmt.Errorf("event_id is required")
	}
	var count int
	if err := tx.QueryRow("SELECT COUNT(*) FROM review_events WHERE id = ?", eventID).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}
