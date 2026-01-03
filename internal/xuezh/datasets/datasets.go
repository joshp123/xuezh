package datasets

import (
	"crypto/sha1"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/joshp123/xuezh/internal/xuezh/clock"
	"github.com/joshp123/xuezh/internal/xuezh/db"
	"github.com/joshp123/xuezh/internal/xuezh/ids"
	"github.com/joshp123/xuezh/internal/xuezh/jsonio"
)

type Row map[string]string

func sha1Hex(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	h := sha1.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func datasetID(datasetType, version string) string {
	payload := datasetType + "|" + version
	h := sha1.Sum([]byte(payload))
	return "ds_" + hex.EncodeToString(h[:])[:12]
}

func readCSV(path string) ([]Row, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	r := csv.NewReader(file)
	headers, err := r.Read()
	if err != nil {
		return nil, err
	}
	rows := []Row{}
	for {
		record, err := r.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		row := Row{}
		anyValue := false
		for i, header := range headers {
			value := ""
			if i < len(record) {
				value = record[i]
			}
			row[header] = value
			if strings.TrimSpace(value) != "" {
				anyValue = true
			}
		}
		if anyValue {
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func parseHSKLevel(value string) (string, error) {
	text := strings.TrimSpace(value)
	text = strings.ReplaceAll(text, "–", "-")
	if text == "7-9" {
		return text, nil
	}
	if _, err := strconv.Atoi(text); err == nil {
		return text, nil
	}
	return "", fmt.Errorf("unsupported hsk_level: %s", value)
}

type executor interface {
	Exec(query string, args ...any) (sql.Result, error)
}

func insertDataset(conn executor, datasetID, datasetType, version, source string) error {
	now, err := clock.NowUTC()
	if err != nil {
		return err
	}
	_, err = conn.Exec(
		`INSERT OR IGNORE INTO datasets (id, dataset_type, version, source, ingested_at)
		 VALUES (?, ?, ?, ?, ?)`,
		datasetID, datasetType, version, source, clock.FormatISO(now),
	)
	return err
}

func insertDatasetItem(conn executor, datasetID, itemID, itemType string, payload map[string]any) error {
	payloadJSON, err := jsonio.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = conn.Exec(
		`INSERT OR IGNORE INTO dataset_items (dataset_id, item_id, item_type, payload_json)
		 VALUES (?, ?, ?, ?)`,
		datasetID, itemID, itemType, payloadJSON,
	)
	return err
}

func ImportDataset(datasetType, path string) (string, int, error) {
	resolved, err := filepath.Abs(expandHome(path))
	if err != nil {
		return "", 0, err
	}
	if _, err := os.Stat(resolved); err != nil {
		return "", 0, fmt.Errorf("dataset file not found: %s", resolved)
	}
	version, err := sha1Hex(resolved)
	if err != nil {
		return "", 0, err
	}
	dsID := datasetID(datasetType, version)
	rows, err := readCSV(resolved)
	if err != nil {
		return "", 0, err
	}
	if datasetType == "frequency" {
		sort.Slice(rows, func(i, j int) bool {
			left, _ := strconv.Atoi(strings.TrimSpace(rows[i]["frequency_rank"]))
			right, _ := strconv.Atoi(strings.TrimSpace(rows[j]["frequency_rank"]))
			return left < right
		})
	}
	dbPath, err := db.InitDB()
	if err != nil {
		return "", 0, err
	}
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return "", 0, err
	}
	defer conn.Close()
	if _, err := conn.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return "", 0, err
	}
	tx, err := conn.Begin()
	if err != nil {
		return "", 0, err
	}
	if err := insertDataset(tx, dsID, datasetType, version, resolved); err != nil {
		return "", 0, err
	}
	now, err := clock.NowUTC()
	if err != nil {
		return "", 0, err
	}
	for idx, row := range rows {
		order := idx + 1
		switch datasetType {
		case "hsk_vocab":
			level, err := parseHSKLevel(row["hsk_level"])
			if err != nil {
				return "", 0, err
			}
			hanzi := row["hanzi"]
			pinyin := row["pinyin"]
			meanings := row["meanings"]
			itemID := ids.WordID(hanzi, pinyin)
			_, err = tx.Exec(
				`INSERT OR IGNORE INTO words (id, hanzi, pinyin, definition, source, created_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				itemID, hanzi, pinyin, meanings, "dataset:hsk_vocab", clock.FormatISO(now),
			)
			if err != nil {
				return "", 0, err
			}
			payload := map[string]any{
				"hsk_level": level,
				"hanzi":     hanzi,
				"pinyin":    pinyin,
				"meanings":  meanings,
				"order":     order,
			}
			if err := insertDatasetItem(tx, dsID, itemID, "word", payload); err != nil {
				return "", 0, err
			}
		case "hsk_chars":
			level, err := parseHSKLevel(row["hsk_level"])
			if err != nil {
				return "", 0, err
			}
			character := row["character"]
			pinyin := row["pinyin"]
			meanings := row["meanings"]
			itemID := ids.CharID(character)
			_, err = tx.Exec(
				`INSERT OR IGNORE INTO characters (id, character, pinyin, definition, source, created_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				itemID, character, pinyin, meanings, "dataset:hsk_chars", clock.FormatISO(now),
			)
			if err != nil {
				return "", 0, err
			}
			payload := map[string]any{
				"hsk_level":    level,
				"character":    character,
				"pinyin":       pinyin,
				"meanings":     meanings,
				"order":        order,
				"radical":      row["radical"],
				"stroke_count": row["stroke_count"],
			}
			if err := insertDatasetItem(tx, dsID, itemID, "character", payload); err != nil {
				return "", 0, err
			}
		case "hsk_grammar":
			level, err := parseHSKLevel(row["hsk_level"])
			if err != nil {
				return "", 0, err
			}
			grammarKey := row["grammar_id"]
			title := row["title"]
			pattern := row["pattern"]
			examples := row["examples"]
			itemID := ids.GrammarID(grammarKey)
			_, err = tx.Exec(
				`INSERT OR IGNORE INTO grammar_points (id, grammar_key, title, notes, source, created_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				itemID, grammarKey, title, pattern, "dataset:hsk_grammar", clock.FormatISO(now),
			)
			if err != nil {
				return "", 0, err
			}
			payload := map[string]any{
				"hsk_level":  level,
				"grammar_id": grammarKey,
				"title":      title,
				"pattern":    pattern,
				"examples":   examples,
				"order":      order,
			}
			if err := insertDatasetItem(tx, dsID, itemID, "grammar", payload); err != nil {
				return "", 0, err
			}
		case "frequency":
			rank, err := strconv.Atoi(strings.TrimSpace(row["frequency_rank"]))
			if err != nil {
				return "", 0, err
			}
			hanzi := row["hanzi"]
			pinyin := row["pinyin"]
			notes := row["notes"]
			itemID := ids.WordID(hanzi, pinyin)
			_, err = tx.Exec(
				`INSERT OR IGNORE INTO words (id, hanzi, pinyin, definition, source, created_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				itemID, hanzi, pinyin, nil, "dataset:frequency", clock.FormatISO(now),
			)
			if err != nil {
				return "", 0, err
			}
			payload := map[string]any{
				"frequency_rank": rank,
				"hanzi":          hanzi,
				"pinyin":         pinyin,
				"notes":          notes,
			}
			if err := insertDatasetItem(tx, dsID, itemID, "word", payload); err != nil {
				return "", 0, err
			}
		default:
			return "", 0, fmt.Errorf("unsupported dataset type: %s", datasetType)
		}
	}
	if err := tx.Commit(); err != nil {
		return "", 0, err
	}
	if err := conn.Close(); err != nil {
		return "", 0, err
	}
	return dsID, len(rows), nil
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			if strings.HasPrefix(path, "~/") {
				return filepath.Join(home, path[2:])
			}
		}
	}
	return path
}
