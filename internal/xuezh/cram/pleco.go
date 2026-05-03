package cram

import (
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/joshp123/xuezh/internal/xuezh/clock"
)

type localSeedItem struct {
	ID            string
	Source        string
	Category      string
	Order         int
	Hanzi         string
	SentenceHanzi string
}

type plecoAssignment struct {
	Card          int
	Hanzi         string
	SentenceHanzi string
}

type seedMatch struct {
	Item localSeedItem
	Card int
}

type plecoScoreRow struct {
	Card              int
	Score             sql.NullInt64
	Difficulty        sql.NullInt64
	History           sql.NullString
	Correct           sql.NullInt64
	Incorrect         sql.NullInt64
	Reviewed          sql.NullInt64
	SinceLastChange   sql.NullInt64
	FirstReviewedTime sql.NullInt64
	LastReviewedTime  sql.NullInt64
	ScoreIncTime      sql.NullInt64
	ScoreDecTime      sql.NullInt64
}

func ImportPlecoScores(opts PlecoScoreImportOptions) (PlecoScoreImportResult, error) {
	path := strings.TrimSpace(opts.Path)
	if path == "" {
		return PlecoScoreImportResult{}, fmt.Errorf("path is required")
	}
	now, err := clock.NowUTC()
	if err != nil {
		return PlecoScoreImportResult{}, err
	}
	nowText := clock.FormatISO(now)

	appDB, err := openDB()
	if err != nil {
		return PlecoScoreImportResult{}, err
	}
	defer appDB.Close()

	plecoDB, err := openPlecoDB(path)
	if err != nil {
		return PlecoScoreImportResult{}, err
	}
	defer plecoDB.Close()

	localItems, err := loadLocalSeedItems(appDB)
	if err != nil {
		return PlecoScoreImportResult{}, err
	}
	categoryIDs, err := loadPlecoCategoryIDs(plecoDB)
	if err != nil {
		return PlecoScoreImportResult{}, err
	}
	scores, err := loadPlecoScores(plecoDB)
	if err != nil {
		return PlecoScoreImportResult{}, err
	}
	settings, err := loadPlecoScoringSettings(plecoDB)
	if err != nil {
		return PlecoScoreImportResult{}, err
	}
	result := PlecoScoreImportResult{
		CanonicalRows: len(localItems),
	}
	grouped := groupSeedItems(localItems)
	tx, err := appDB.Begin()
	if err != nil {
		return PlecoScoreImportResult{}, err
	}
	defer tx.Rollback()

	for _, key := range sortedGroupKeys(grouped) {
		items := grouped[key]
		catID, ok := categoryIDs[key]
		if !ok {
			result.UnseededRows += len(items)
			result.UnseededCategories = append(result.UnseededCategories, categoryLabel(key))
			continue
		}
		assignments, err := plecoAssignments(plecoDB, catID)
		if err != nil {
			return result, err
		}
		matches := matchSeedAssignments(items, assignments)
		if len(matches) != len(items) {
			result.UnseededRows += len(items) - len(matches)
			result.UnseededCategories = append(result.UnseededCategories, categoryLabel(key))
		}
		result.SeededRows += len(matches)
		for _, match := range matches {
			score := scores[match.Card]
			seed := seedFromPlecoScore(score, settings, now)
			if err := updateSeedState(tx, match.Item.ID, match.Card, score, seed, nowText); err != nil {
				return result, err
			}
		}
	}
	if err := storeScoringSettings(tx, settings, nowText); err != nil {
		return result, err
	}
	if err := tx.Commit(); err != nil {
		return result, err
	}
	return result, nil
}

func loadPlecoScoringSettings(conn *sql.DB) (scoringSettings, error) {
	settings := defaultScoringSettings()
	err := conn.QueryRow("SELECT id, name FROM pleco_flash_profiles WHERE name = 'Spaced Repetition'").
		Scan(&settings.ProfileID, &settings.ProfileName)
	if err == sql.ErrNoRows {
		err = conn.QueryRow("SELECT id, name FROM pleco_flash_profiles ORDER BY sort, id LIMIT 1").
			Scan(&settings.ProfileID, &settings.ProfileName)
	}
	if err != nil {
		return scoringSettings{}, err
	}
	rows, err := conn.Query("SELECT propid, propvalue FROM pleco_flash_profilesettings WHERE propset = ?", settings.ProfileID)
	if err != nil {
		return scoringSettings{}, err
	}
	defer rows.Close()
	props := map[string]string{}
	for rows.Next() {
		var key string
		var value sql.NullString
		if err := rows.Scan(&key, &value); err != nil {
			return scoringSettings{}, err
		}
		if value.Valid {
			props[key] = value.String
		}
	}
	if err := rows.Err(); err != nil {
		return scoringSettings{}, err
	}
	settings.LearnedScore = profileInt(props, "pro_learnedselcount", settings.LearnedScore)
	settings.MinScore = profileInt(props, "pro_scoreautomin", settings.MinScore)
	settings.MaxScore = profileInt(props, "pro_scoreautomax", settings.MaxScore)
	settings.MinDifficulty = profileInt(props, "pro_scoremindifficulty", settings.MinDifficulty)
	settings.MaxDifficulty = profileInt(props, "pro_scoremaxdifficulty", settings.MaxDifficulty)
	settings.PointsPerDay = profileInt(props, "pro_cardpointsday", settings.PointsPerDay)
	settings.ScoreOncePerDay = profileInt(props, "pro_scoreonceaday", 1) != 0
	settings.IncorrectScore = profileInt(props, "pro_scoredecreaseamt", settings.IncorrectScore)
	settings.CorrectInitialScore = profileInt(props, "pro_scoreinitinterval6", settings.CorrectInitialScore)
	settings.CorrectMultiplier = profileInt(props, "pro_scoreintervalmult6", settings.CorrectMultiplier)
	settings.CorrectDifficultyChange = profileInt(props, "pro_scorediffchange6", settings.CorrectDifficultyChange)
	settings.IncorrectDifficultyChange = profileInt(props, "pro_scorediffchange2", settings.IncorrectDifficultyChange)
	settings.DifficultyDivisor = profileInt(props, "pro_scorediffdivisor", settings.DifficultyDivisor)
	return normalizeScoringSettings(settings), nil
}

func profileInt(props map[string]string, key string, fallback int) int {
	raw, ok := props[key]
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return value
}

func openPlecoDB(path string) (*sql.DB, error) {
	abs, err := filepath.Abs(expandHome(path))
	if err != nil {
		return nil, err
	}
	plecoURL := url.URL{Scheme: "file", Path: abs, RawQuery: "mode=ro"}
	dsn := plecoURL.String()
	conn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Exec("PRAGMA query_only = ON;"); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

func loadLocalSeedItems(conn *sql.DB) ([]localSeedItem, error) {
	rows, err := conn.Query(`
		SELECT id, source, category, learning_order, hanzi, sentence_hanzi
		FROM cram_items
		ORDER BY source, category, learning_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []localSeedItem{}
	for rows.Next() {
		var item localSeedItem
		if err := rows.Scan(&item.ID, &item.Source, &item.Category, &item.Order, &item.Hanzi, &item.SentenceHanzi); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func loadPlecoCategoryIDs(conn *sql.DB) (map[string]int, error) {
	roots := map[string]string{
		"HelloChinese":    SourceHelloChinese,
		"Travel Survival": SourceTravelSurvival,
	}
	categoryIDs := map[string]int{}
	for rootName, source := range roots {
		var rootID int
		err := conn.QueryRow("SELECT id FROM pleco_flash_categories WHERE name = ? AND parent = -2", rootName).Scan(&rootID)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return nil, err
		}
		rows, err := conn.Query("SELECT id, name FROM pleco_flash_categories WHERE parent = ? ORDER BY sort, id", rootID)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id int
			var name string
			if err := rows.Scan(&id, &name); err != nil {
				rows.Close()
				return nil, err
			}
			categoryIDs[groupKey(source, strings.TrimSpace(name))] = id
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}
	return categoryIDs, nil
}

func loadPlecoScores(conn *sql.DB) (map[int]plecoScoreRow, error) {
	rows, err := conn.Query(`
		SELECT card, score, difficulty, history, correct, incorrect, reviewed,
		       sincelastchange, firstreviewedtime, lastreviewedtime,
		       scoreinctime, scoredectime
		FROM pleco_flash_scores_1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	scores := map[int]plecoScoreRow{}
	for rows.Next() {
		var row plecoScoreRow
		if err := rows.Scan(
			&row.Card, &row.Score, &row.Difficulty, &row.History,
			&row.Correct, &row.Incorrect, &row.Reviewed,
			&row.SinceLastChange, &row.FirstReviewedTime, &row.LastReviewedTime,
			&row.ScoreIncTime, &row.ScoreDecTime,
		); err != nil {
			return nil, err
		}
		scores[row.Card] = row
	}
	return scores, rows.Err()
}

func plecoAssignments(conn *sql.DB, categoryID int) ([]plecoAssignment, error) {
	rows, err := conn.Query(`
		SELECT a.card, c.hw, c.defn
		FROM pleco_flash_categoryassigns a
		JOIN pleco_flash_cards c ON c.id = a.card
		WHERE a.cat = ?
		ORDER BY a.id`, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	assignments := []plecoAssignment{}
	for rows.Next() {
		var assignment plecoAssignment
		var hanzi, defn sql.NullString
		if err := rows.Scan(&assignment.Card, &hanzi, &defn); err != nil {
			return nil, err
		}
		if hanzi.Valid {
			assignment.Hanzi = normalizeMatchText(hanzi.String)
		}
		if defn.Valid {
			assignment.SentenceHanzi = plecoDefnSentence(defn.String)
		}
		assignments = append(assignments, assignment)
	}
	return assignments, rows.Err()
}

func matchSeedAssignments(items []localSeedItem, assignments []plecoAssignment) []seedMatch {
	if len(items) == len(assignments) {
		matches := make([]seedMatch, 0, len(items))
		for i, item := range items {
			matches = append(matches, seedMatch{Item: item, Card: assignments[i].Card})
		}
		return matches
	}

	matches := []seedMatch{}
	usedItems := map[int]bool{}
	usedAssignments := map[int]bool{}
	byText := map[string][]int{}
	for i, assignment := range assignments {
		key := matchKey(assignment.Hanzi, assignment.SentenceHanzi)
		if key != "" {
			byText[key] = append(byText[key], i)
		}
	}
	for i, item := range items {
		key := matchKey(normalizeMatchText(item.Hanzi), normalizeMatchText(item.SentenceHanzi))
		candidates := byText[key]
		if len(candidates) == 1 && !usedAssignments[candidates[0]] {
			usedItems[i] = true
			usedAssignments[candidates[0]] = true
			matches = append(matches, seedMatch{Item: item, Card: assignments[candidates[0]].Card})
		}
	}

	itemWordCounts := map[string]int{}
	assignmentWords := map[string][]int{}
	for i, item := range items {
		if !usedItems[i] {
			itemWordCounts[normalizeMatchText(item.Hanzi)]++
		}
	}
	for i, assignment := range assignments {
		if !usedAssignments[i] {
			assignmentWords[assignment.Hanzi] = append(assignmentWords[assignment.Hanzi], i)
		}
	}
	for i, item := range items {
		word := normalizeMatchText(item.Hanzi)
		candidates := assignmentWords[word]
		if !usedItems[i] && itemWordCounts[word] == 1 && len(candidates) == 1 {
			usedItems[i] = true
			usedAssignments[candidates[0]] = true
			matches = append(matches, seedMatch{Item: item, Card: assignments[candidates[0]].Card})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		return matches[i].Item.Order < matches[j].Item.Order
	})
	return matches
}

func matchKey(hanzi, sentence string) string {
	if hanzi == "" || sentence == "" {
		return ""
	}
	return hanzi + "\x1f" + sentence
}

func plecoDefnSentence(defn string) string {
	fields := strings.Split(defn, "\ueab1")
	if len(fields) < 3 {
		return ""
	}
	return normalizeMatchText(fields[2])
}

func normalizeMatchText(text string) string {
	text = strings.ReplaceAll(text, "@", "")
	text = strings.ReplaceAll(text, "\u200b", "")
	return cleanChineseSentence(text)
}

type stateSeed struct {
	NextDueAt any
}

func seedFromPlecoScore(score plecoScoreRow, settings scoringSettings, now time.Time) stateSeed {
	var nextDue any
	if score.Score.Valid && score.Score.Int64 > 0 && score.LastReviewedTime.Valid && score.LastReviewedTime.Int64 > 0 {
		interval := intervalMinutesForScore(int(score.Score.Int64), settings)
		nextDue = clock.FormatISO(time.Unix(score.LastReviewedTime.Int64, 0).UTC().Add(time.Duration(interval) * time.Minute))
	}
	return stateSeed{NextDueAt: nextDue}
}

func updateSeedState(tx *sql.Tx, itemID string, cardID int, score plecoScoreRow, seed stateSeed, nowText string) error {
	var currentLastReviewed sql.NullString
	if err := tx.QueryRow("SELECT pleco_last_reviewed_at FROM cram_state WHERE item_id = ?", itemID).Scan(&currentLastReviewed); err != nil {
		return err
	}
	if currentLastReviewed.Valid && score.LastReviewedTime.Valid && score.LastReviewedTime.Int64 > 0 {
		current, err := time.Parse(time.RFC3339, currentLastReviewed.String)
		if err == nil && current.After(time.Unix(score.LastReviewedTime.Int64, 0).UTC()) {
			return nil
		}
	}
	_, err := tx.Exec(`
		UPDATE cram_state
		SET next_due_at = ?,
		    pleco_card_id = ?, pleco_score = ?, pleco_difficulty = ?, pleco_history = ?,
		    pleco_correct_count = ?, pleco_incorrect_count = ?, pleco_reviewed_count = ?,
		    pleco_first_reviewed_at = ?, pleco_last_reviewed_at = ?,
		    pleco_sincelastchange = ?, pleco_score_inc_time = ?, pleco_score_dec_time = ?,
		    score_imported_at = ?, updated_at = ?
		WHERE item_id = ?`,
		seed.NextDueAt,
		cardID, nullInt64Value(score.Score), nullInt64Value(score.Difficulty), nullStringValue(score.History),
		nullInt64Value(score.Correct), nullInt64Value(score.Incorrect), nullInt64Value(score.Reviewed),
		plecoTimeValue(score.FirstReviewedTime), plecoTimeValue(score.LastReviewedTime),
		nullInt64Value(score.SinceLastChange), nullInt64Value(score.ScoreIncTime), nullInt64Value(score.ScoreDecTime),
		nowText, nowText, itemID)
	return err
}

func groupSeedItems(items []localSeedItem) map[string][]localSeedItem {
	grouped := map[string][]localSeedItem{}
	for _, item := range items {
		key := groupKey(item.Source, item.Category)
		grouped[key] = append(grouped[key], item)
	}
	for key := range grouped {
		sort.Slice(grouped[key], func(i, j int) bool {
			return grouped[key][i].Order < grouped[key][j].Order
		})
	}
	return grouped
}

func sortedGroupKeys(grouped map[string][]localSeedItem) []string {
	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func groupKey(source, category string) string {
	return source + "\x1f" + category
}

func categoryLabel(key string) string {
	parts := strings.SplitN(key, "\x1f", 2)
	if len(parts) != 2 {
		return key
	}
	return parts[0] + " / " + parts[1]
}

func nullInt64Value(value sql.NullInt64) any {
	if value.Valid {
		return value.Int64
	}
	return nil
}

func nullStringValue(value sql.NullString) any {
	if value.Valid {
		return value.String
	}
	return nil
}

func plecoTimeValue(value sql.NullInt64) any {
	if value.Valid && value.Int64 > 0 {
		return clock.FormatISO(time.Unix(value.Int64, 0).UTC())
	}
	return nil
}
