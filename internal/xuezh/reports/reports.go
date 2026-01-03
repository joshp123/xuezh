package reports

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/joshp123/xuezh/internal/xuezh/clock"
	"github.com/joshp123/xuezh/internal/xuezh/db"
	"github.com/joshp123/xuezh/internal/xuezh/envelope"
	"github.com/joshp123/xuezh/internal/xuezh/jsonio"
	"github.com/joshp123/xuezh/internal/xuezh/paths"
)

type ReportResult struct {
	Data      map[string]any
	Artifacts []envelope.Artifact
	Truncated bool
	Limits    map[string]any
}

func artifactPath(prefix string, now time.Time) (string, error) {
	root, err := paths.EnsureWorkspace()
	if err != nil {
		return "", err
	}
	dayPath := filepath.Join(root, "artifacts", now.Format("2006"), now.Format("01"), now.Format("02"))
	if err := os.MkdirAll(dayPath, 0o755); err != nil {
		return "", err
	}
	filename := fmt.Sprintf("%s-%s.json", prefix, now.UTC().Format("20060102T150405Z"))
	return filepath.Join(dayPath, filename), nil
}

func latestDatasetID(conn *sql.DB, datasetType string) (string, error) {
	row := conn.QueryRow("SELECT id FROM datasets WHERE dataset_type = ? ORDER BY ingested_at DESC LIMIT 1", datasetType)
	var id string
	err := row.Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return id, nil
}

func countsByLevel(conn *sql.DB, datasetID string) (map[string]int, error) {
	rows, err := conn.Query("SELECT payload_json FROM dataset_items WHERE dataset_id = ?", datasetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var payloadJSON string
		if err := rows.Scan(&payloadJSON); err != nil {
			return nil, err
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
			return nil, err
		}
		level := fmt.Sprintf("%v", payload["hsk_level"])
		counts[level] = counts[level] + 1
	}
	return counts, rows.Err()
}

func knownItems(conn *sql.DB, itemType string) (map[string]struct{}, error) {
	var rows *sql.Rows
	var err error
	if itemType != "" {
		rows, err = conn.Query("SELECT item_id FROM user_knowledge WHERE item_type = ? AND seen_count > 0", itemType)
	} else {
		rows, err = conn.Query("SELECT item_id FROM user_knowledge WHERE seen_count > 0")
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	known := map[string]struct{}{}
	for rows.Next() {
		var itemID string
		if err := rows.Scan(&itemID); err != nil {
			return nil, err
		}
		known[itemID] = struct{}{}
	}
	return known, rows.Err()
}

func levelRange(level string) (int, int, bool) {
	value := strings.TrimSpace(level)
	value = strings.ReplaceAll(value, "–", "-")
	if value == "7-9" {
		return 7, 9, true
	}
	if n, err := strconv.Atoi(value); err == nil {
		return n, n, true
	}
	if strings.Contains(value, "-") {
		parts := strings.Split(value, "-")
		if len(parts) == 2 {
			start, errStart := strconv.Atoi(parts[0])
			end, errEnd := strconv.Atoi(parts[1])
			if errStart == nil && errEnd == nil {
				return start, end, true
			}
		}
	}
	return 0, 0, false
}

func levelSortKey(level string) (int, int, string) {
	start, end, ok := levelRange(level)
	if !ok {
		return 999, 999, level
	}
	return start, end, level
}

func countsByLevelStats(items []itemLevel, knownIDs map[string]struct{}) map[string]map[string]any {
	levels := map[string]struct{}{}
	for _, item := range items {
		levels[item.Level] = struct{}{}
	}
	levelList := make([]string, 0, len(levels))
	for level := range levels {
		levelList = append(levelList, level)
	}
	sort.Slice(levelList, func(i, j int) bool {
		ai0, ai1, ai2 := levelSortKey(levelList[i])
		aj0, aj1, aj2 := levelSortKey(levelList[j])
		if ai0 != aj0 {
			return ai0 < aj0
		}
		if ai1 != aj1 {
			return ai1 < aj1
		}
		return ai2 < aj2
	})
	byLevel := map[string]map[string]any{}
	for _, level := range levelList {
		total := 0
		known := 0
		for _, item := range items {
			if item.Level != level {
				continue
			}
			total++
			if _, ok := knownIDs[item.ItemID]; ok {
				known++
			}
		}
		unknown := total - known
		coverage := 0.0
		if total > 0 {
			coverage = float64(known) / float64(total)
		}
		byLevel[level] = map[string]any{
			"total":        total,
			"known":        known,
			"unknown":      unknown,
			"coverage_pct": floatNumber(coverage),
		}
	}
	return byLevel
}

type itemLevel struct {
	ItemID string
	Level  string
}

func evidenceRows(items []itemLevel, knownIDs map[string]struct{}, maxItems int) []map[string]any {
	sorted := make([]itemLevel, len(items))
	copy(sorted, items)
	sort.Slice(sorted, func(i, j int) bool {
		ai0, ai1, ai2 := levelSortKey(sorted[i].Level)
		aj0, aj1, aj2 := levelSortKey(sorted[j].Level)
		if ai0 != aj0 {
			return ai0 < aj0
		}
		if ai1 != aj1 {
			return ai1 < aj1
		}
		if ai2 != aj2 {
			return ai2 < aj2
		}
		return sorted[i].ItemID < sorted[j].ItemID
	})
	rows := []map[string]any{}
	for _, item := range sorted {
		status := "unknown"
		if _, ok := knownIDs[item.ItemID]; ok {
			status = "known"
		}
		if status == "unknown" {
			rows = append(rows, map[string]any{"item_id": item.ItemID, "level": item.Level, "status": status})
		}
		if len(rows) >= maxItems {
			break
		}
	}
	return rows
}

func spillIfNeeded(envelopeData map[string]any, maxBytes int, prefix string) (map[string]any, []envelope.Artifact, bool, error) {
	payload, err := jsonio.Dumps(envelopeData)
	if err != nil {
		return nil, nil, false, err
	}
	if len([]byte(payload)) <= maxBytes {
		return envelopeData, nil, false, nil
	}
	now, err := clock.NowUTC()
	if err != nil {
		return nil, nil, false, err
	}
	spillPath, err := artifactPath(prefix, now)
	if err != nil {
		return nil, nil, false, err
	}
	if err := os.WriteFile(spillPath, []byte(payload), 0o644); err != nil {
		return nil, nil, false, err
	}
	workspace, err := paths.EnsureWorkspace()
	if err != nil {
		return nil, nil, false, err
	}
	relPath, err := filepath.Rel(workspace, spillPath)
	if err != nil {
		return nil, nil, false, err
	}
	stat, err := os.Stat(spillPath)
	if err != nil {
		return nil, nil, false, err
	}
	artifact := envelope.Artifact{Path: relPath, MIME: "application/json", Purpose: prefix + "_spill", Bytes: intPtr(int(stat.Size()))}
	truncated := map[string]any{}
	for k, v := range envelopeData {
		truncated[k] = v
	}
	data, _ := truncated["data"].(map[string]any)
	truncated["data"] = map[string]any{
		"spill_artifact": relPath,
		"window":         data["window"],
		"level":          data["level"],
		"item_type":      data["item_type"],
	}
	truncated["artifacts"] = []envelope.Artifact{artifact}
	truncated["truncated"] = true
	return truncated, []envelope.Artifact{artifact}, true, nil
}

func BuildHSKReport(level, window string, maxItems, maxBytes int, includeChars bool) (ReportResult, error) {
	dbPath, err := db.InitDB()
	if err != nil {
		return ReportResult{}, err
	}
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return ReportResult{}, err
	}
	defer conn.Close()
	vocabID, err := latestDatasetID(conn, "hsk_vocab")
	if err != nil {
		return ReportResult{}, err
	}
	grammarID, err := latestDatasetID(conn, "hsk_grammar")
	if err != nil {
		return ReportResult{}, err
	}
	charsID := ""
	if includeChars {
		charsID, err = latestDatasetID(conn, "hsk_chars")
		if err != nil {
			return ReportResult{}, err
		}
	}
	vocabItems := []itemLevel{}
	grammarItems := []itemLevel{}
	charsItems := []itemLevel{}
	if vocabID != "" {
		rows, err := conn.Query("SELECT item_id, payload_json FROM dataset_items WHERE dataset_id = ?", vocabID)
		if err != nil {
			return ReportResult{}, err
		}
		for rows.Next() {
			var itemID, payloadJSON string
			if err := rows.Scan(&itemID, &payloadJSON); err != nil {
				rows.Close()
				return ReportResult{}, err
			}
			var payload map[string]any
			if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
				rows.Close()
				return ReportResult{}, err
			}
			levelValue := strings.TrimSpace(fmt.Sprintf("%v", payload["hsk_level"]))
			vocabItems = append(vocabItems, itemLevel{ItemID: itemID, Level: levelValue})
		}
		rows.Close()
	}
	if grammarID != "" {
		rows, err := conn.Query("SELECT item_id, payload_json FROM dataset_items WHERE dataset_id = ?", grammarID)
		if err != nil {
			return ReportResult{}, err
		}
		for rows.Next() {
			var itemID, payloadJSON string
			if err := rows.Scan(&itemID, &payloadJSON); err != nil {
				rows.Close()
				return ReportResult{}, err
			}
			var payload map[string]any
			if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
				rows.Close()
				return ReportResult{}, err
			}
			levelValue := strings.TrimSpace(fmt.Sprintf("%v", payload["hsk_level"]))
			grammarItems = append(grammarItems, itemLevel{ItemID: itemID, Level: levelValue})
		}
		rows.Close()
	}
	if charsID != "" {
		rows, err := conn.Query("SELECT item_id, payload_json FROM dataset_items WHERE dataset_id = ?", charsID)
		if err != nil {
			return ReportResult{}, err
		}
		for rows.Next() {
			var itemID, payloadJSON string
			if err := rows.Scan(&itemID, &payloadJSON); err != nil {
				rows.Close()
				return ReportResult{}, err
			}
			var payload map[string]any
			if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
				rows.Close()
				return ReportResult{}, err
			}
			levelValue := strings.TrimSpace(fmt.Sprintf("%v", payload["hsk_level"]))
			charsItems = append(charsItems, itemLevel{ItemID: itemID, Level: levelValue})
		}
		rows.Close()
	}
	knownVocab, err := knownItems(conn, "word")
	if err != nil {
		return ReportResult{}, err
	}
	knownGrammar, err := knownItems(conn, "grammar")
	if err != nil {
		return ReportResult{}, err
	}
	knownChars, err := knownItems(conn, "character")
	if err != nil {
		return ReportResult{}, err
	}
	vocabLevels := countsByLevelStats(vocabItems, knownVocab)
	grammarLevels := countsByLevelStats(grammarItems, knownGrammar)
	charsLevels := countsByLevelStats(charsItems, knownChars)
	filterLevels := func(items []itemLevel) []itemLevel {
		levelValue := strings.TrimSpace(level)
		if levelValue == "7-9" {
			filtered := []itemLevel{}
			for _, item := range items {
				if item.Level == "7-9" {
					filtered = append(filtered, item)
				}
			}
			return filtered
		}
		if strings.Contains(levelValue, "-") {
			parts := strings.Split(levelValue, "-")
			if len(parts) == 2 {
				minLevel, errMin := strconv.Atoi(parts[0])
				maxLevel, errMax := strconv.Atoi(parts[1])
				if errMin == nil && errMax == nil {
					filtered := []itemLevel{}
					for _, item := range items {
						rangeStart, rangeEnd, ok := levelRange(item.Level)
						if !ok {
							continue
						}
						if rangeStart >= minLevel && rangeEnd <= maxLevel {
							filtered = append(filtered, item)
						}
					}
					return filtered
				}
			}
		}
		if _, err := strconv.Atoi(levelValue); err == nil {
			maxLevel, _ := strconv.Atoi(levelValue)
			filtered := []itemLevel{}
			for _, item := range items {
				rangeStart, _, ok := levelRange(item.Level)
				if !ok {
					continue
				}
				if rangeStart <= maxLevel {
					filtered = append(filtered, item)
				}
			}
			return filtered
		}
		return items
	}
	vocabItems = filterLevels(vocabItems)
	grammarItems = filterLevels(grammarItems)
	charsItems = filterLevels(charsItems)
	coverage := map[string]map[string]any{
		"vocab": {
			"total": len(vocabItems),
			"known": countKnown(vocabItems, knownVocab),
		},
		"grammar": {
			"total": len(grammarItems),
			"known": countKnown(grammarItems, knownGrammar),
		},
	}
	coverage["vocab"]["unknown"] = coverage["vocab"]["total"].(int) - coverage["vocab"]["known"].(int)
	coverage["vocab"]["coverage_pct"] = pct(coverage["vocab"]["known"].(int), coverage["vocab"]["total"].(int))
	coverage["grammar"]["unknown"] = coverage["grammar"]["total"].(int) - coverage["grammar"]["known"].(int)
	coverage["grammar"]["coverage_pct"] = pct(coverage["grammar"]["known"].(int), coverage["grammar"]["total"].(int))
	if includeChars {
		coverage["chars"] = map[string]any{
			"total": len(charsItems),
			"known": countKnown(charsItems, knownChars),
		}
		coverage["chars"]["unknown"] = coverage["chars"]["total"].(int) - coverage["chars"]["known"].(int)
		coverage["chars"]["coverage_pct"] = pct(coverage["chars"]["known"].(int), coverage["chars"]["total"].(int))
	}
	data := map[string]any{
		"level":    level,
		"window":   window,
		"coverage": coverage,
		"evidence": evidenceRows(vocabItems, knownVocab, maxItems),
		"counts_by_level": map[string]any{
			"vocab":   vocabLevels,
			"grammar": grammarLevels,
		},
	}
	if includeChars {
		data["counts_by_level"].(map[string]any)["chars"] = charsLevels
	}
	env := map[string]any{
		"ok":             true,
		"schema_version": "1",
		"command":        "report.hsk",
		"data":           data,
		"artifacts":      []envelope.Artifact{},
		"truncated":      false,
		"limits": map[string]any{
			"max_items": maxItems,
			"max_bytes": maxBytes,
		},
	}
	env, artifacts, truncated, err := spillIfNeeded(env, maxBytes, "report-hsk")
	if err != nil {
		return ReportResult{}, err
	}
	return ReportResult{Data: dataFromEnvelope(env), Artifacts: artifacts, Truncated: truncated, Limits: limitsFromEnvelope(env)}, nil
}

func BuildMasteryReport(itemType, window string, maxItems, maxBytes int) (ReportResult, error) {
	now, err := clock.NowUTC()
	if err != nil {
		return ReportResult{}, err
	}
	duration, err := clock.ParseDuration(window)
	if err != nil {
		return ReportResult{}, err
	}
	since := now.Add(-duration)
	dbPath, err := db.InitDB()
	if err != nil {
		return ReportResult{}, err
	}
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return ReportResult{}, err
	}
	defer conn.Close()
	rows, err := conn.Query(`
		SELECT
		  item_id,
		  last_seen_at,
		  seen_count,
		  COALESCE(recall_due_at, due_at),
		  COALESCE(recall_last_grade, last_grade),
		  pronunciation_due_at,
		  pronunciation_last_grade
		FROM user_knowledge
		WHERE item_type = ? AND last_seen_at IS NOT NULL AND last_seen_at >= ?
		ORDER BY last_seen_at DESC, item_id ASC
		`, itemType, clock.FormatISO(since))
	if err != nil {
		return ReportResult{}, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var itemID, lastSeen, recallDueAt, pronunciationDueAt sql.NullString
		var seenCount sql.NullInt64
		var recallLastGrade, pronunciationLastGrade sql.NullInt64
		if err := rows.Scan(&itemID, &lastSeen, &seenCount, &recallDueAt, &recallLastGrade, &pronunciationDueAt, &pronunciationLastGrade); err != nil {
			return ReportResult{}, err
		}
		items = append(items, map[string]any{
			"item_id":                  itemID.String,
			"last_seen":                lastSeen.String,
			"times_seen":               int(seenCount.Int64),
			"recall_due_at":            nullString(recallDueAt),
			"recall_last_grade":        nullInt(recallLastGrade),
			"pronunciation_due_at":     nullString(pronunciationDueAt),
			"pronunciation_last_grade": nullInt(pronunciationLastGrade),
		})
		if len(items) >= maxItems {
			break
		}
	}
	data := map[string]any{"item_type": itemType, "window": window, "items": items}
	env := map[string]any{
		"ok":             true,
		"schema_version": "1",
		"command":        "report.mastery",
		"data":           data,
		"artifacts":      []envelope.Artifact{},
		"truncated":      false,
		"limits": map[string]any{
			"max_items": maxItems,
			"max_bytes": maxBytes,
		},
	}
	env, artifacts, truncated, err := spillIfNeeded(env, maxBytes, "report-mastery")
	if err != nil {
		return ReportResult{}, err
	}
	return ReportResult{Data: dataFromEnvelope(env), Artifacts: artifacts, Truncated: truncated, Limits: limitsFromEnvelope(env)}, nil
}

func dataFromEnvelope(env map[string]any) map[string]any {
	data, _ := env["data"].(map[string]any)
	if data == nil {
		return map[string]any{}
	}
	return data
}

func limitsFromEnvelope(env map[string]any) map[string]any {
	limits, _ := env["limits"].(map[string]any)
	if limits == nil {
		return map[string]any{}
	}
	return limits
}

func countKnown(items []itemLevel, known map[string]struct{}) int {
	count := 0
	for _, item := range items {
		if _, ok := known[item.ItemID]; ok {
			count++
		}
	}
	return count
}

func pct(known, total int) json.Number {
	if total == 0 {
		return json.Number("0.0")
	}
	return floatNumber(float64(known) / float64(total))
}

func floatNumber(value float64) json.Number {
	if math.Abs(value-math.Round(value)) < 1e-9 {
		return json.Number(fmt.Sprintf("%.1f", value))
	}
	return json.Number(strconv.FormatFloat(value, 'g', -1, 64))
}

func nullString(value sql.NullString) any {
	if value.Valid {
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

func intPtr(value int) *int {
	return &value
}
