package hellochinese

import (
	"bufio"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"

	"github.com/joshp123/xuezh/internal/xuezh/audio"
	"github.com/joshp123/xuezh/internal/xuezh/clock"
	"github.com/joshp123/xuezh/internal/xuezh/db"
	"github.com/joshp123/xuezh/internal/xuezh/paths"
)

var DefaultVoices = []string{
	"zh-CN-XiaoxiaoNeural",
	"zh-CN-XiaoyiNeural",
	"zh-CN-YunxiNeural",
	"zh-CN-YunyangNeural",
}

type AudioGenerator func(text, voice, outPath string) (string, error)

type ImportOptions struct {
	Path           string
	AudioMode      string
	Voices         []string
	AudioGenerator AudioGenerator
}

type ImportResult struct {
	RowsSeen       int `json:"rows_seen"`
	RowsInserted   int `json:"rows_inserted"`
	RowsExisting   int `json:"rows_existing"`
	AudioGenerated int `json:"audio_generated"`
	AudioExisting  int `json:"audio_existing"`
	AudioFailed    int `json:"audio_failed"`
}

type Card struct {
	ItemID             string            `json:"item_id"`
	LearningOrder      int               `json:"learning_order"`
	Word               string            `json:"word"`
	Pinyin             string            `json:"pinyin"`
	Meaning            string            `json:"meaning"`
	SentenceHanzi      string            `json:"sentence_hanzi"`
	SentencePinyin     string            `json:"sentence_pinyin"`
	SentenceMeaning    string            `json:"sentence_meaning"`
	SentenceAudioPaths map[string]string `json:"sentence_audio_paths"`
	Status             string            `json:"status"`
	DueAt              *string           `json:"due_at"`
	UnknownOtherWords  *int              `json:"unknown_other_words"`
}

type GradeResult struct {
	ItemID     string `json:"item_id"`
	Grade      string `json:"grade"`
	Status     string `json:"status"`
	NextDueAt  string `json:"next_due_at"`
	SeenCount  int    `json:"seen_count"`
	LapseCount int    `json:"lapse_count"`
}

type AudioBackfillOptions struct {
	Voices         []string
	Concurrency    int
	Limit          int
	AudioGenerator AudioGenerator
}

type AudioBackfillResult struct {
	TasksSeen      int `json:"tasks_seen"`
	AudioGenerated int `json:"audio_generated"`
	AudioExisting  int `json:"audio_existing"`
	AudioFailed    int `json:"audio_failed"`
}

type inputRow struct {
	Index           int    `json:"index"`
	UnitLabel       string `json:"unit_label"`
	Pinyin          string `json:"pinyin"`
	Hanzi           string `json:"hanzi"`
	Meaning         string `json:"meaning"`
	SentencePinyin  string `json:"sentence_pinyin"`
	SentenceHanzi   string `json:"sentence_hanzi"`
	SentenceMeaning string `json:"sentence_meaning"`
}

type itemRow struct {
	ID                 string
	LearningOrder      int
	SourceIndex        int
	UnitLabel          string
	Pinyin             string
	Hanzi              string
	Meaning            string
	SentencePinyin     string
	SentenceHanzi      string
	SentenceHanziRaw   string
	SentenceMeaning    string
	RowHash            string
	SentenceAudioPaths map[string]string
}

func ImportCorpus(opts ImportOptions) (ImportResult, error) {
	if strings.TrimSpace(opts.Path) == "" {
		return ImportResult{}, fmt.Errorf("path is required")
	}
	audioMode := opts.AudioMode
	if audioMode == "" {
		audioMode = "none"
	}
	if audioMode != "none" && audioMode != "sentence" {
		return ImportResult{}, fmt.Errorf("unsupported audio mode: %s", audioMode)
	}
	voices := opts.Voices
	if len(voices) == 0 {
		voices = DefaultVoices
	}
	gen := opts.AudioGenerator
	if gen == nil {
		gen = generateSentenceAudio
	}
	conn, err := openDB()
	if err != nil {
		return ImportResult{}, err
	}
	defer conn.Close()

	file, err := os.Open(expandHome(opts.Path))
	if err != nil {
		return ImportResult{}, err
	}
	defer file.Close()

	now, err := clock.NowUTC()
	if err != nil {
		return ImportResult{}, err
	}
	nowText := clock.FormatISO(now)
	result := ImportResult{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		result.RowsSeen++
		row, err := parseRow(scanner.Bytes(), result.RowsSeen)
		if err != nil {
			return result, err
		}
		item, err := upsertItem(conn, row, nowText)
		if err != nil {
			return result, err
		}
		if item.inserted {
			result.RowsInserted++
		} else {
			result.RowsExisting++
		}
		if audioMode == "sentence" {
			audioResult, err := ensureAudio(conn, item.item, voices, gen, nowText)
			result.AudioGenerated += audioResult.generated
			result.AudioExisting += audioResult.existing
			if err != nil {
				result.AudioFailed++
				return result, err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return result, err
	}
	return result, nil
}

func NextCards(limit int, now time.Time) ([]Card, error) {
	if limit <= 0 {
		limit = 1
	}
	conn, err := openDB()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	rows, err := conn.Query(`
		SELECT i.id, i.learning_order, i.hanzi, i.pinyin, i.meaning,
		       i.sentence_hanzi, i.sentence_pinyin, i.sentence_meaning,
		       i.sentence_audio_paths_json, s.status, s.next_due_at
		FROM hellochinese_items i
		JOIN hellochinese_cram_state s ON s.item_id = i.id
		WHERE s.next_due_at IS NULL OR s.next_due_at <= ?
		ORDER BY
		  CASE WHEN s.next_due_at IS NULL THEN 1 ELSE 0 END,
		  COALESCE(s.next_due_at, ''),
		  i.learning_order
		LIMIT ?`, clock.FormatISO(now), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cards := []Card{}
	for rows.Next() {
		var card Card
		var due sql.NullString
		var audioJSON string
		if err := rows.Scan(&card.ItemID, &card.LearningOrder, &card.Word, &card.Pinyin, &card.Meaning, &card.SentenceHanzi, &card.SentencePinyin, &card.SentenceMeaning, &audioJSON, &card.Status, &due); err != nil {
			return nil, err
		}
		if due.Valid {
			card.DueAt = &due.String
		}
		card.SentenceAudioPaths = map[string]string{}
		_ = json.Unmarshal([]byte(audioJSON), &card.SentenceAudioPaths)
		cards = append(cards, card)
	}
	return cards, rows.Err()
}

func GradeCard(itemID string, grade string, now time.Time) (GradeResult, error) {
	itemID = strings.TrimSpace(itemID)
	if itemID == "" {
		return GradeResult{}, fmt.Errorf("item is required")
	}
	delay, status, lapse, err := gradePolicy(grade)
	if err != nil {
		return GradeResult{}, err
	}
	conn, err := openDB()
	if err != nil {
		return GradeResult{}, err
	}
	defer conn.Close()
	nextDue := clock.FormatISO(now.Add(delay))
	nowText := clock.FormatISO(now)
	res, err := conn.Exec(`
		UPDATE hellochinese_cram_state
		SET status = ?, next_due_at = ?, seen_count = seen_count + 1,
		    lapse_count = lapse_count + ?, last_grade = ?, updated_at = ?
		WHERE item_id = ?`, status, nextDue, lapse, grade, nowText, itemID)
	if err != nil {
		return GradeResult{}, err
	}
	changed, err := res.RowsAffected()
	if err != nil {
		return GradeResult{}, err
	}
	if changed == 0 {
		return GradeResult{}, fmt.Errorf("item not found: %s", itemID)
	}
	payload, _ := json.Marshal(map[string]any{"mode": "hellochinese_cram", "grade": grade, "next_due_at": nextDue})
	_, _ = conn.Exec(
		"INSERT INTO review_events (id, item_id, event_type, ts, payload_json) VALUES (?, ?, ?, ?, ?)",
		uuid.NewString(), itemID, "cram.grade", nowText, string(payload),
	)
	var result GradeResult
	row := conn.QueryRow("SELECT item_id, status, next_due_at, seen_count, lapse_count, last_grade FROM hellochinese_cram_state WHERE item_id = ?", itemID)
	if err := row.Scan(&result.ItemID, &result.Status, &result.NextDueAt, &result.SeenCount, &result.LapseCount, &result.Grade); err != nil {
		return GradeResult{}, err
	}
	return result, nil
}

func BackfillAudio(opts AudioBackfillOptions) (AudioBackfillResult, error) {
	voices := opts.Voices
	if len(voices) == 0 {
		voices = DefaultVoices
	}
	concurrency := opts.Concurrency
	if concurrency <= 0 {
		concurrency = 4
	}
	gen := opts.AudioGenerator
	if gen == nil {
		gen = generateSentenceAudio
	}
	conn, err := openDB()
	if err != nil {
		return AudioBackfillResult{}, err
	}
	defer conn.Close()
	items, err := listItemsForAudio(conn, opts.Limit)
	if err != nil {
		return AudioBackfillResult{}, err
	}
	type task struct {
		item  itemRow
		voice string
	}
	tasks := make(chan task)
	var generated atomic.Int64
	var existing atomic.Int64
	var failed atomic.Int64
	var firstErr error
	var errMu sync.Mutex
	var writeMu sync.Mutex
	var wg sync.WaitGroup
	now, err := clock.NowUTC()
	if err != nil {
		return AudioBackfillResult{}, err
	}
	nowText := clock.FormatISO(now)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range tasks {
				if path := task.item.SentenceAudioPaths[task.voice]; path != "" && workspaceFileExists(path) {
					existing.Add(1)
					continue
				}
				outPath := filepath.ToSlash(filepath.Join("artifacts", "hellochinese", "sentences", task.item.ID, task.voice+".ogg"))
				artifactPath, err := gen(task.item.SentenceHanzi, task.voice, outPath)
				if err == nil {
					writeMu.Lock()
					err = mergeAudioPath(conn, task.item.ID, task.voice, artifactPath, nowText)
					writeMu.Unlock()
				}
				if err != nil {
					failed.Add(1)
					errMu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					errMu.Unlock()
					continue
				}
				generated.Add(1)
			}
		}()
	}
	tasksSeen := 0
	for _, item := range items {
		for _, voice := range voices {
			voice = strings.TrimSpace(voice)
			if voice == "" {
				continue
			}
			tasksSeen++
			tasks <- task{item: item, voice: voice}
		}
	}
	close(tasks)
	wg.Wait()
	result := AudioBackfillResult{
		TasksSeen:      tasksSeen,
		AudioGenerated: int(generated.Load()),
		AudioExisting:  int(existing.Load()),
		AudioFailed:    int(failed.Load()),
	}
	if firstErr != nil {
		return result, firstErr
	}
	return result, nil
}

func listItemsForAudio(conn *sql.DB, limit int) ([]itemRow, error) {
	query := `
		SELECT id, learning_order, source_index, unit_label, pinyin, hanzi, meaning,
		       sentence_pinyin, sentence_hanzi, sentence_hanzi_raw, sentence_meaning,
		       row_hash, sentence_audio_paths_json
		FROM hellochinese_items
		ORDER BY learning_order`
	args := []any{}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}
	rows, err := conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []itemRow{}
	for rows.Next() {
		var item itemRow
		var source sql.NullInt64
		var audioJSON string
		if err := rows.Scan(&item.ID, &item.LearningOrder, &source, &item.UnitLabel, &item.Pinyin, &item.Hanzi, &item.Meaning, &item.SentencePinyin, &item.SentenceHanzi, &item.SentenceHanziRaw, &item.SentenceMeaning, &item.RowHash, &audioJSON); err != nil {
			return nil, err
		}
		if source.Valid {
			item.SourceIndex = int(source.Int64)
		}
		item.SentenceAudioPaths = map[string]string{}
		_ = json.Unmarshal([]byte(audioJSON), &item.SentenceAudioPaths)
		items = append(items, item)
	}
	return items, rows.Err()
}

func parseRow(line []byte, lineNumber int) (itemRow, error) {
	var in inputRow
	if err := json.Unmarshal(line, &in); err != nil {
		return itemRow{}, fmt.Errorf("line %d: %w", lineNumber, err)
	}
	order := in.Index
	if order <= 0 {
		order = lineNumber
	}
	sentence := cleanChineseSentence(in.SentenceHanzi)
	if strings.TrimSpace(in.Hanzi) == "" || strings.TrimSpace(sentence) == "" {
		return itemRow{}, fmt.Errorf("line %d: hanzi and sentence_hanzi are required", lineNumber)
	}
	row := itemRow{
		LearningOrder:    order,
		SourceIndex:      in.Index,
		UnitLabel:        strings.TrimSpace(in.UnitLabel),
		Pinyin:           strings.TrimSpace(in.Pinyin),
		Hanzi:            strings.TrimSpace(in.Hanzi),
		Meaning:          strings.TrimSpace(in.Meaning),
		SentencePinyin:   strings.TrimSpace(in.SentencePinyin),
		SentenceHanzi:    sentence,
		SentenceHanziRaw: strings.TrimSpace(in.SentenceHanzi),
		SentenceMeaning:  strings.TrimSpace(in.SentenceMeaning),
	}
	row.RowHash = hashRow(row)
	return row, nil
}

type upsertResult struct {
	item     itemRow
	inserted bool
}

func upsertItem(conn *sql.DB, row itemRow, nowText string) (upsertResult, error) {
	existing, found, err := findExisting(conn, row.RowHash)
	if err != nil {
		return upsertResult{}, err
	}
	if found {
		return upsertResult{item: existing}, nil
	}
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM hellochinese_items WHERE learning_order = ? OR (hanzi = ? AND sentence_hanzi = ?)", row.LearningOrder, row.Hanzi, row.SentenceHanzi).Scan(&count); err != nil {
		return upsertResult{}, err
	}
	if count > 0 {
		return upsertResult{}, fmt.Errorf("conflicting HelloChinese row at learning_order %d", row.LearningOrder)
	}
	row.ID = uuid.NewString()
	row.SentenceAudioPaths = map[string]string{}
	audioJSON := "{}"
	_, err = conn.Exec(`
		INSERT INTO hellochinese_items (
		  id, learning_order, source_index, unit_label, pinyin, hanzi, meaning,
		  sentence_pinyin, sentence_hanzi, sentence_hanzi_raw, sentence_meaning,
		  row_hash, sentence_audio_paths_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		row.ID, row.LearningOrder, nullableInt(row.SourceIndex), row.UnitLabel, row.Pinyin, row.Hanzi, row.Meaning,
		row.SentencePinyin, row.SentenceHanzi, row.SentenceHanziRaw, row.SentenceMeaning,
		row.RowHash, audioJSON, nowText, nowText)
	if err != nil {
		return upsertResult{}, err
	}
	_, err = conn.Exec(
		"INSERT INTO hellochinese_cram_state (item_id, status, created_at, updated_at) VALUES (?, ?, ?, ?)",
		row.ID, "new", nowText, nowText,
	)
	if err != nil {
		return upsertResult{}, err
	}
	return upsertResult{item: row, inserted: true}, nil
}

func findExisting(conn *sql.DB, rowHash string) (itemRow, bool, error) {
	var row itemRow
	var source sql.NullInt64
	var audioJSON string
	err := conn.QueryRow(`
		SELECT id, learning_order, source_index, unit_label, pinyin, hanzi, meaning,
		       sentence_pinyin, sentence_hanzi, sentence_hanzi_raw, sentence_meaning,
		       row_hash, sentence_audio_paths_json
		FROM hellochinese_items WHERE row_hash = ?`, rowHash).
		Scan(&row.ID, &row.LearningOrder, &source, &row.UnitLabel, &row.Pinyin, &row.Hanzi, &row.Meaning, &row.SentencePinyin, &row.SentenceHanzi, &row.SentenceHanziRaw, &row.SentenceMeaning, &row.RowHash, &audioJSON)
	if err == sql.ErrNoRows {
		return itemRow{}, false, nil
	}
	if err != nil {
		return itemRow{}, false, err
	}
	if source.Valid {
		row.SourceIndex = int(source.Int64)
	}
	row.SentenceAudioPaths = map[string]string{}
	_ = json.Unmarshal([]byte(audioJSON), &row.SentenceAudioPaths)
	return row, true, nil
}

type audioEnsureResult struct {
	generated int
	existing  int
}

func ensureAudio(conn *sql.DB, item itemRow, voices []string, gen AudioGenerator, nowText string) (audioEnsureResult, error) {
	result := audioEnsureResult{}
	audioPaths := item.SentenceAudioPaths
	if audioPaths == nil {
		audioPaths = map[string]string{}
	}
	for _, voice := range voices {
		voice = strings.TrimSpace(voice)
		if voice == "" {
			continue
		}
		if existing := audioPaths[voice]; existing != "" && workspaceFileExists(existing) {
			result.existing++
			continue
		}
		outPath := filepath.ToSlash(filepath.Join("artifacts", "hellochinese", "sentences", item.ID, voice+".ogg"))
		artifactPath, err := gen(item.SentenceHanzi, voice, outPath)
		if err != nil {
			return result, err
		}
		if err := mergeAudioPath(conn, item.ID, voice, artifactPath, nowText); err != nil {
			return result, err
		}
		result.generated++
	}
	return result, nil
}

func mergeAudioPath(conn *sql.DB, itemID, voice, path, nowText string) error {
	tx, err := conn.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	var audioJSON string
	if err := tx.QueryRow("SELECT sentence_audio_paths_json FROM hellochinese_items WHERE id = ?", itemID).Scan(&audioJSON); err != nil {
		return err
	}
	audioPaths := map[string]string{}
	_ = json.Unmarshal([]byte(audioJSON), &audioPaths)
	audioPaths[voice] = path
	encoded, err := json.Marshal(audioPaths)
	if err != nil {
		return err
	}
	if _, err := tx.Exec("UPDATE hellochinese_items SET sentence_audio_paths_json = ?, updated_at = ? WHERE id = ?", string(encoded), nowText, itemID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func generateSentenceAudio(text, voice, outPath string) (string, error) {
	result, err := audio.TTSAudio(text, voice, outPath, "edge-tts", "hellochinese_sentence_tts")
	if err != nil {
		return "", err
	}
	if len(result.Artifacts) == 0 {
		return "", fmt.Errorf("tts generated no artifact")
	}
	return result.Artifacts[0].Path, nil
}

func gradePolicy(grade string) (time.Duration, string, int, error) {
	switch grade {
	case "again":
		return 0, "learning", 1, nil
	case "hard":
		return 10 * time.Minute, "learning", 0, nil
	case "good":
		return 2 * time.Hour, "learning", 0, nil
	case "easy":
		return 24 * time.Hour, "review", 0, nil
	default:
		return 0, "", 0, fmt.Errorf("invalid grade: %s", grade)
	}
}

func openDB() (*sql.DB, error) {
	if _, err := db.InitDB(); err != nil {
		return nil, err
	}
	dbPath, err := paths.DBPath()
	if err != nil {
		return nil, err
	}
	resolved, err := paths.ResolveInWorkspace(dbPath)
	if err != nil {
		return nil, err
	}
	conn, err := sql.Open("sqlite3", resolved)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if _, err := conn.Exec("PRAGMA busy_timeout = 30000;"); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

func cleanChineseSentence(text string) string {
	text = strings.TrimSpace(text)
	var b strings.Builder
	var prevHan bool
	for _, r := range text {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if prevHan {
				continue
			}
			b.WriteRune(r)
			prevHan = false
			continue
		}
		b.WriteRune(r)
		prevHan = isHan(r)
	}
	return strings.TrimSpace(b.String())
}

func isHan(r rune) bool {
	return r >= 0x4E00 && r <= 0x9FFF
}

func hashRow(row itemRow) string {
	payload := map[string]any{
		"learning_order":   row.LearningOrder,
		"source_index":     row.SourceIndex,
		"unit_label":       row.UnitLabel,
		"pinyin":           row.Pinyin,
		"hanzi":            row.Hanzi,
		"meaning":          row.Meaning,
		"sentence_pinyin":  row.SentencePinyin,
		"sentence_hanzi":   row.SentenceHanzi,
		"sentence_meaning": row.SentenceMeaning,
	}
	body, _ := json.Marshal(payload)
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func workspaceFileExists(rel string) bool {
	resolved, err := paths.ResolveInWorkspace(rel)
	if err != nil {
		return false
	}
	info, err := os.Stat(resolved)
	return err == nil && !info.IsDir()
}

func nullableInt(value int) any {
	if value == 0 {
		return nil
	}
	return value
}

func expandHome(path string) string {
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}
