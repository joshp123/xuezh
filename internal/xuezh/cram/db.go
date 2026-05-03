package cram

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"

	"github.com/joshp123/xuezh/internal/xuezh/db"
	"github.com/joshp123/xuezh/internal/xuezh/paths"
)

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

func upsertItem(conn *sql.DB, row itemRow, nowText string) (itemRow, bool, error) {
	existing, found, err := findExisting(conn, row.RowHash)
	if err != nil {
		return itemRow{}, false, err
	}
	if found {
		return existing, false, nil
	}
	existing, found, err = findExistingByNaturalKey(conn, row, nowText)
	if err != nil {
		return itemRow{}, false, err
	}
	if found {
		return existing, false, nil
	}
	var count int
	err = conn.QueryRow(
		"SELECT COUNT(*) FROM cram_items WHERE source = ? AND (learning_order = ? OR (hanzi = ? AND sentence_hanzi = ?))",
		row.Source, row.LearningOrder, row.Hanzi, row.SentenceHanzi,
	).Scan(&count)
	if err != nil {
		return itemRow{}, false, err
	}
	if count > 0 {
		return itemRow{}, false, fmt.Errorf("conflicting cram row for %s at learning_order %d", row.Source, row.LearningOrder)
	}
	row.ID = newID()
	row.SentenceAudioPaths = map[string]string{}
	audioJSON := "{}"
	_, err = conn.Exec(`
		INSERT INTO cram_items (
		  id, source, category, learning_order, source_index, pinyin, hanzi, meaning,
		  sentence_pinyin, sentence_hanzi, sentence_hanzi_raw, sentence_meaning,
		  row_hash, sentence_audio_paths_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		row.ID, row.Source, row.Category, row.LearningOrder, nullableInt(row.SourceIndex),
		row.Pinyin, row.Hanzi, row.Meaning, row.SentencePinyin, row.SentenceHanzi,
		row.SentenceHanziRaw, row.SentenceMeaning, row.RowHash, audioJSON, nowText, nowText)
	if err != nil {
		return itemRow{}, false, err
	}
	_, err = conn.Exec(
		"INSERT INTO cram_state (item_id, created_at, updated_at) VALUES (?, ?, ?)",
		row.ID, nowText, nowText,
	)
	if err != nil {
		return itemRow{}, false, err
	}
	return row, true, nil
}

func findExistingByNaturalKey(conn *sql.DB, row itemRow, nowText string) (itemRow, bool, error) {
	existing, found, err := findOne(conn, `
		SELECT id, source, category, learning_order, source_index, pinyin, hanzi, meaning,
		       sentence_pinyin, sentence_hanzi, sentence_hanzi_raw, sentence_meaning,
		       row_hash, sentence_audio_paths_json
		FROM cram_items WHERE source = ? AND learning_order = ?`, row.Source, row.LearningOrder)
	if err != nil {
		return itemRow{}, false, err
	}
	if found {
		updated, err := updateExistingItem(conn, existing, row, nowText)
		return updated, true, err
	}
	existing, found, err = findOne(conn, `
		SELECT id, source, category, learning_order, source_index, pinyin, hanzi, meaning,
		       sentence_pinyin, sentence_hanzi, sentence_hanzi_raw, sentence_meaning,
		       row_hash, sentence_audio_paths_json
		FROM cram_items WHERE source = ? AND hanzi = ? AND sentence_hanzi = ?`, row.Source, row.Hanzi, row.SentenceHanzi)
	if err != nil || !found {
		return existing, found, err
	}
	return itemRow{}, false, fmt.Errorf("conflicting cram row for %s at learning_order %d", row.Source, row.LearningOrder)
}

func updateExistingItem(conn *sql.DB, existing, row itemRow, nowText string) (itemRow, error) {
	row.ID = existing.ID
	row.SentenceAudioPaths = existing.SentenceAudioPaths
	audioJSON, err := json.Marshal(row.SentenceAudioPaths)
	if err != nil {
		return itemRow{}, err
	}
	_, err = conn.Exec(`
		UPDATE cram_items
		SET category = ?, source_index = ?, pinyin = ?, hanzi = ?, meaning = ?,
		    sentence_pinyin = ?, sentence_hanzi = ?, sentence_hanzi_raw = ?,
		    sentence_meaning = ?, row_hash = ?, sentence_audio_paths_json = ?,
		    updated_at = ?
		WHERE id = ?`,
		row.Category, nullableInt(row.SourceIndex), row.Pinyin, row.Hanzi, row.Meaning,
		row.SentencePinyin, row.SentenceHanzi, row.SentenceHanziRaw,
		row.SentenceMeaning, row.RowHash, string(audioJSON), nowText, row.ID,
	)
	if err != nil {
		return itemRow{}, err
	}
	return row, nil
}

func findExisting(conn *sql.DB, rowHash string) (itemRow, bool, error) {
	return findOne(conn, `
		SELECT id, source, category, learning_order, source_index, pinyin, hanzi, meaning,
		       sentence_pinyin, sentence_hanzi, sentence_hanzi_raw, sentence_meaning,
		       row_hash, sentence_audio_paths_json
		FROM cram_items WHERE row_hash = ?`, rowHash)
}

func findOne(conn *sql.DB, query string, args ...any) (itemRow, bool, error) {
	var row itemRow
	var source sql.NullInt64
	var audioJSON string
	err := conn.QueryRow(query, args...).
		Scan(&row.ID, &row.Source, &row.Category, &row.LearningOrder, &source,
			&row.Pinyin, &row.Hanzi, &row.Meaning, &row.SentencePinyin, &row.SentenceHanzi,
			&row.SentenceHanziRaw, &row.SentenceMeaning, &row.RowHash, &audioJSON)
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

func hashRow(row itemRow) string {
	payload := strings.Join([]string{
		row.Source,
		row.Category,
		fmt.Sprintf("%d", row.LearningOrder),
		row.Pinyin,
		row.Hanzi,
		row.Meaning,
		row.SentencePinyin,
		row.SentenceHanzi,
		row.SentenceMeaning,
	}, "\x1f")
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func cleanChineseSentence(text string) string {
	text = strings.TrimSpace(text)
	var b strings.Builder
	var previousCJK bool
	for _, r := range text {
		if unicode.IsSpace(r) {
			continue
		}
		if r == '?' {
			r = '？'
		}
		if r == '!' {
			r = '！'
		}
		currentCJK := isCJK(r)
		if b.Len() > 0 && !previousCJK && !currentCJK && r != '。' && r != '，' && r != '？' && r != '！' {
			b.WriteRune(' ')
		}
		b.WriteRune(r)
		previousCJK = currentCJK
	}
	return b.String()
}

func isCJK(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || (r >= 0x3400 && r <= 0x4DBF)
}

func sourceLabel(source string) string {
	switch source {
	case SourceHelloChinese:
		return "HelloChinese"
	case SourceTravelSurvival:
		return "Travel Survival"
	default:
		return source
	}
}

func expandHome(path string) string {
	if path == "" || path[0] != '~' {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if len(path) == 1 {
		return home
	}
	if path[1] == '/' || path[1] == '\\' {
		return filepath.Join(home, path[2:])
	}
	return path
}

func nullableInt(value int) any {
	if value == 0 {
		return nil
	}
	return value
}

func newID() string {
	return uuid.NewString()
}

func workspaceFileExists(path string) bool {
	if path == "" {
		return false
	}
	fullPath, err := paths.ResolveInWorkspace(path)
	if err != nil {
		return false
	}
	info, err := os.Stat(fullPath)
	return err == nil && !info.IsDir()
}
