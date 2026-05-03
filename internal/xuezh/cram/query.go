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

type itemScanner interface {
	Scan(dest ...any) error
}

func OverviewFor(now time.Time) (Overview, error) {
	conn, err := openDB()
	if err != nil {
		return Overview{}, err
	}
	defer conn.Close()
	nowText := clock.FormatISO(now)
	overview := Overview{
		GeneratedAt: nowText,
		Sources:     []SourceSummary{},
		Categories:  []CategorySummary{},
	}
	sourceRows, err := conn.Query(`
		SELECT i.source,
		       COUNT(*) AS total,
		       SUM(CASE WHEN s.pleco_score >= COALESCE((SELECT learned_score FROM cram_pleco_scoring_settings WHERE id = 1), 200) THEN 1 ELSE 0 END) AS learned,
		       SUM(CASE
		         WHEN s.pleco_score IS NULL THEN 1
		         WHEN s.pleco_score < COALESCE((SELECT learned_score FROM cram_pleco_scoring_settings WHERE id = 1), 200) THEN 1
		         WHEN s.next_due_at IS NOT NULL AND s.next_due_at <= ? THEN 1
		         ELSE 0
		       END) AS eligible
		FROM cram_items i
		JOIN cram_state s ON s.item_id = i.id
		GROUP BY i.source
		ORDER BY i.source`, nowText)
	if err != nil {
		return Overview{}, err
	}
	defer sourceRows.Close()
	for sourceRows.Next() {
		var row SourceSummary
		if err := sourceRows.Scan(&row.Source, &row.TotalCount, &row.LearnedCount, &row.Eligible); err != nil {
			return Overview{}, err
		}
		row.Label = sourceLabel(row.Source)
		overview.Sources = append(overview.Sources, row)
	}
	if err := sourceRows.Err(); err != nil {
		return Overview{}, err
	}
	categoryRows, err := conn.Query(`
		SELECT i.source, i.category,
		       COUNT(*) AS total,
		       SUM(CASE WHEN s.pleco_score >= COALESCE((SELECT learned_score FROM cram_pleco_scoring_settings WHERE id = 1), 200) THEN 1 ELSE 0 END) AS learned,
		       SUM(CASE
		         WHEN s.pleco_score IS NULL THEN 1
		         WHEN s.pleco_score < COALESCE((SELECT learned_score FROM cram_pleco_scoring_settings WHERE id = 1), 200) THEN 1
		         WHEN s.next_due_at IS NOT NULL AND s.next_due_at <= ? THEN 1
		         ELSE 0
		       END) AS eligible
		FROM cram_items i
		JOIN cram_state s ON s.item_id = i.id
		GROUP BY i.source, i.category
		ORDER BY i.source, MIN(i.learning_order)`, nowText)
	if err != nil {
		return Overview{}, err
	}
	defer categoryRows.Close()
	for categoryRows.Next() {
		var row CategorySummary
		if err := categoryRows.Scan(&row.Source, &row.Category, &row.TotalCount, &row.LearnedCount, &row.Eligible); err != nil {
			return Overview{}, err
		}
		row.SourceLabel = sourceLabel(row.Source)
		overview.Categories = append(overview.Categories, row)
	}
	return overview, categoryRows.Err()
}

func NextCards(opts NextOptions, now time.Time) ([]Card, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 1
	}
	conn, err := openDB()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	where, args, dueFilter := nextWhere(opts)
	queryArgs := append([]any{}, args...)
	if dueFilter {
		queryArgs = append(queryArgs, clock.FormatISO(now))
	}
	queryArgs = append(queryArgs, clock.FormatISO(now), limit)
	rows, err := conn.Query(fmt.Sprintf(`
		SELECT i.id, i.source, i.category, i.learning_order, i.source_index, i.pinyin, i.hanzi, i.meaning,
		       i.sentence_pinyin, i.sentence_hanzi, i.sentence_hanzi_raw, i.sentence_meaning,
		       i.row_hash, i.sentence_audio_paths_json, s.next_due_at
		FROM cram_items i
		JOIN cram_state s ON s.item_id = i.id
		%s
		%s
		ORDER BY
		  CASE WHEN s.pleco_score IS NULL OR s.pleco_score < COALESCE((SELECT learned_score FROM cram_pleco_scoring_settings WHERE id = 1), 200) THEN 0 ELSE 1 END,
		  CASE WHEN s.next_due_at IS NOT NULL AND s.next_due_at <= ? THEN 0 ELSE 1 END,
		  COALESCE(s.pleco_incorrect_count, 0) DESC,
		  CASE WHEN s.pleco_score IS NULL THEN 0 ELSE 1 END,
		  COALESCE(s.pleco_score, 999999),
		  i.source,
		  i.learning_order
		LIMIT ?`, where, dueClause(dueFilter)), queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cards := []Card{}
	for rows.Next() {
		var item itemRow
		var sourceIndex sql.NullInt64
		var audioJSON string
		var due sql.NullString
		if err := rows.Scan(
			&item.ID, &item.Source, &item.Category, &item.LearningOrder, &sourceIndex,
			&item.Pinyin, &item.Hanzi, &item.Meaning, &item.SentencePinyin,
			&item.SentenceHanzi, &item.SentenceHanziRaw, &item.SentenceMeaning,
			&item.RowHash, &audioJSON, &due,
		); err != nil {
			return nil, err
		}
		if sourceIndex.Valid {
			item.SourceIndex = int(sourceIndex.Int64)
		}
		item.SentenceAudioPaths = map[string]string{}
		_ = json.Unmarshal([]byte(audioJSON), &item.SentenceAudioPaths)
		card := cardFromItem(item)
		if due.Valid {
			card.DueAt = &due.String
		}
		cards = append(cards, card)
	}
	return cards, rows.Err()
}

func PracticePreviewFor(filters PracticeFilters, now time.Time) (PracticePreview, error) {
	filters = normalizePracticeFilters(filters)
	conn, err := openDB()
	if err != nil {
		return PracticePreview{}, err
	}
	defer conn.Close()
	settings, err := loadLocalScoringSettings(conn)
	if err != nil {
		return PracticePreview{}, err
	}
	categorySet := map[string]bool{}
	for _, category := range filters.Categories {
		source := strings.TrimSpace(category.Source)
		name := strings.TrimSpace(category.Category)
		if source != "" && name != "" {
			categorySet[source+"\x00"+name] = true
		}
	}
	rows, err := conn.Query(`
		SELECT i.id, i.source, i.category, i.learning_order, i.hanzi, i.sentence_hanzi,
		       s.pleco_score, s.pleco_correct_count, s.pleco_incorrect_count,
		       s.pleco_reviewed_count, s.pleco_last_reviewed_at, s.next_due_at
		FROM cram_items i
		JOIN cram_state s ON s.item_id = i.id
		ORDER BY i.source, i.learning_order`)
	if err != nil {
		return PracticePreview{}, err
	}
	defer rows.Close()

	sourceMap := map[string]*PracticeSourceSummary{}
	categoryMap := map[string]*PracticeCategorySummary{}
	categoryOrder := []string{}
	cards := []PracticeCard{}
	for rows.Next() {
		var card PracticeCard
		var score sql.NullInt64
		var correct sql.NullInt64
		var incorrect sql.NullInt64
		var reviewed sql.NullInt64
		var lastReviewed sql.NullString
		var nextDue sql.NullString
		if err := rows.Scan(
			&card.ItemID, &card.Source, &card.Category, &card.LearningOrder,
			&card.Word, &card.SentenceHanzi, &score, &correct, &incorrect,
			&reviewed, &lastReviewed, &nextDue,
		); err != nil {
			return PracticePreview{}, err
		}
		card.SourceLabel = sourceLabel(card.Source)
		if score.Valid {
			value := int(score.Int64)
			card.Score = &value
		}
		if correct.Valid {
			card.CorrectCount = int(correct.Int64)
		}
		if incorrect.Valid {
			card.IncorrectCount = int(incorrect.Int64)
		}
		if reviewed.Valid {
			card.ReviewedCount = int(reviewed.Int64)
		}
		if lastReviewed.Valid && strings.TrimSpace(lastReviewed.String) != "" {
			value := lastReviewed.String
			card.LastReviewedAt = &value
		}
		card.NotLearned = practiceNotLearned(card.Score, filters)
		learnedProgressMissing := scoreBelowLearnedThreshold(card.Score, settings)
		card.Due = practiceDue(nextDue, now)
		card.GotWrong = card.IncorrectCount > filters.MissesMoreThan
		matches := practiceMatches(card, filters)
		if len(categorySet) > 0 && !categorySet[card.Source+"\x00"+card.Category] {
			matches = false
		}
		source := sourceMap[card.Source]
		if source == nil {
			source = &PracticeSourceSummary{Source: card.Source, Label: sourceLabel(card.Source)}
			sourceMap[card.Source] = source
		}
		source.TotalCount++
		if learnedProgressMissing {
			source.NotLearnedCount++
		}
		if card.Due {
			source.DueCount++
		}
		if card.GotWrong {
			source.GotWrongCount++
		}
		addScoreBucket(&source.ScoreBuckets, card.Score)
		categoryKey := card.Source + "\x00" + card.Category
		category := categoryMap[categoryKey]
		if category == nil {
			category = &PracticeCategorySummary{
				Source:      card.Source,
				SourceLabel: sourceLabel(card.Source),
				Category:    card.Category,
			}
			categoryMap[categoryKey] = category
			categoryOrder = append(categoryOrder, categoryKey)
		}
		category.TotalCount++
		if learnedProgressMissing {
			category.NotLearnedCount++
		}
		if card.Due {
			category.DueCount++
		}
		if card.GotWrong {
			category.GotWrongCount++
		}
		addScoreBucket(&category.ScoreBuckets, card.Score)
		if matches {
			source.PracticeCount++
			category.PracticeCount++
			cards = append(cards, card)
		}
	}
	if err := rows.Err(); err != nil {
		return PracticePreview{}, err
	}
	sort.SliceStable(cards, func(i, j int) bool {
		return practiceCardLess(cards[i], cards[j])
	})
	preview := PracticePreview{
		GeneratedAt: clock.FormatISO(now),
		Sources:     []PracticeSourceSummary{},
		Categories:  []PracticeCategorySummary{},
		Cards:       cards,
	}
	for _, source := range []string{SourceHelloChinese, SourceTravelSurvival} {
		if row := sourceMap[source]; row != nil {
			preview.Sources = append(preview.Sources, *row)
		}
	}
	for _, key := range categoryOrder {
		preview.Categories = append(preview.Categories, *categoryMap[key])
	}
	return preview, nil
}

func scanItemRow(row itemScanner) (itemRow, error) {
	var item itemRow
	var sourceIndex sql.NullInt64
	var audioJSON string
	err := row.Scan(
		&item.ID, &item.Source, &item.Category, &item.LearningOrder, &sourceIndex,
		&item.Pinyin, &item.Hanzi, &item.Meaning, &item.SentencePinyin,
		&item.SentenceHanzi, &item.SentenceHanziRaw, &item.SentenceMeaning,
		&item.RowHash, &audioJSON,
	)
	if err != nil {
		return itemRow{}, err
	}
	if sourceIndex.Valid {
		item.SourceIndex = int(sourceIndex.Int64)
	}
	item.SentenceAudioPaths = map[string]string{}
	_ = json.Unmarshal([]byte(audioJSON), &item.SentenceAudioPaths)
	return item, nil
}

func nextWhere(opts NextOptions) (string, []any, bool) {
	cardIDs := []string{}
	for _, id := range opts.CardIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			cardIDs = append(cardIDs, id)
		}
	}
	if len(cardIDs) > 0 {
		placeholders := strings.TrimRight(strings.Repeat("?,", len(cardIDs)), ",")
		args := make([]any, 0, len(cardIDs))
		for _, id := range cardIDs {
			args = append(args, id)
		}
		return "WHERE i.id IN (" + placeholders + ")", args, false
	}
	return "WHERE 1 = 1", nil, true
}

func dueClause(enabled bool) string {
	if !enabled {
		return ""
	}
	return `AND (
		s.pleco_score IS NULL
		OR s.pleco_score < COALESCE((SELECT learned_score FROM cram_pleco_scoring_settings WHERE id = 1), 200)
		OR s.next_due_at <= ?
	)`
}

func normalizePracticeFilters(filters PracticeFilters) PracticeFilters {
	if filters.ScoreBelow <= 0 {
		filters.ScoreBelow = 200
	}
	if filters.MissesMoreThan < 0 {
		filters.MissesMoreThan = 0
	}
	return filters
}

func practiceNotLearned(score *int, filters PracticeFilters) bool {
	if score == nil {
		return filters.IncludeNoScore
	}
	return *score < filters.ScoreBelow
}

func scoreBelowLearnedThreshold(score *int, settings scoringSettings) bool {
	settings = normalizeScoringSettings(settings)
	return score == nil || *score < settings.LearnedScore
}

func practiceMatches(card PracticeCard, filters PracticeFilters) bool {
	if !filters.IncludeNotLearned && !filters.IncludeDue && !filters.IncludeGotWrong {
		return true
	}
	return (filters.IncludeNotLearned && card.NotLearned) ||
		(filters.IncludeDue && card.Due) ||
		(filters.IncludeGotWrong && card.GotWrong)
}

func practiceCardLess(left, right PracticeCard) bool {
	if left.NotLearned != right.NotLearned {
		return left.NotLearned
	}
	if left.Due != right.Due {
		return left.Due
	}
	if left.IncorrectCount != right.IncorrectCount {
		return left.IncorrectCount > right.IncorrectCount
	}
	leftNoScore := left.Score == nil
	rightNoScore := right.Score == nil
	if leftNoScore != rightNoScore {
		return leftNoScore
	}
	leftScore := 1<<31 - 1
	if left.Score != nil {
		leftScore = *left.Score
	}
	rightScore := 1<<31 - 1
	if right.Score != nil {
		rightScore = *right.Score
	}
	if leftScore != rightScore {
		return leftScore < rightScore
	}
	if left.Source != right.Source {
		return left.Source < right.Source
	}
	return left.LearningOrder < right.LearningOrder
}

func practiceDue(value sql.NullString, now time.Time) bool {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return false
	}
	parsed, err := time.Parse(time.RFC3339, value.String)
	if err != nil {
		return false
	}
	return !parsed.After(now)
}

func addScoreBucket(buckets *ScoreBuckets, score *int) {
	if score == nil {
		buckets.NoScore++
		return
	}
	switch {
	case *score < 100:
		buckets.ScoreUnder100++
	case *score < 200:
		buckets.Score100To199++
	case *score < 600:
		buckets.Score200To599++
	case *score < 1600:
		buckets.Score600To1599++
	case *score < 6400:
		buckets.Score1600To6399++
	default:
		buckets.Score6400Plus++
	}
}

func cardFromItem(item itemRow) Card {
	return Card{
		ItemID:             item.ID,
		Source:             item.Source,
		Category:           item.Category,
		LearningOrder:      item.LearningOrder,
		Word:               item.Hanzi,
		Pinyin:             item.Pinyin,
		Meaning:            item.Meaning,
		SentenceHanzi:      item.SentenceHanzi,
		SentencePinyin:     item.SentencePinyin,
		SentenceMeaning:    item.SentenceMeaning,
		SentenceAudioPaths: item.SentenceAudioPaths,
	}
}
