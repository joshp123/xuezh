package cram

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/joshp123/xuezh/internal/xuezh/audio"
	"github.com/joshp123/xuezh/internal/xuezh/clock"
)

type audioEnsureResult struct {
	generated int
	existing  int
}

func BackfillAudio(opts AudioBackfillOptions) (AudioBackfillResult, error) {
	voices := opts.Voices
	if len(voices) == 0 {
		voices = DefaultVoices
	}
	voiceRates := opts.VoiceRates
	if len(voiceRates) == 0 {
		voiceRates = DefaultVoiceRates
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
	items, err := listItemsForAudio(conn, opts.Source, opts.Limit)
	if err != nil {
		return AudioBackfillResult{}, err
	}
	now, err := clock.NowUTC()
	if err != nil {
		return AudioBackfillResult{}, err
	}
	nowText := clock.FormatISO(now)
	if opts.Replace {
		if err := clearAudioPaths(conn, opts.Source, nowText); err != nil {
			return AudioBackfillResult{}, err
		}
		for i := range items {
			items[i].SentenceAudioPaths = map[string]string{}
		}
	}
	type task struct {
		item  itemRow
		voice string
		rate  string
	}
	tasks := make(chan task)
	var generated atomic.Int64
	var existing atomic.Int64
	var failed atomic.Int64
	var firstErr error
	var errMu sync.Mutex
	var writeMu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range tasks {
				if path := task.item.SentenceAudioPaths[task.voice]; path != "" && workspaceFileExists(path) {
					existing.Add(1)
					continue
				}
				outPath := filepath.ToSlash(filepath.Join("artifacts", "cram", task.item.Source, "sentences", task.item.ID, task.voice+".ogg"))
				artifactPath, err := gen(task.item.SentenceHanzi, task.voice, task.rate, outPath)
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
			tasks <- task{item: item, voice: voice, rate: voiceRates[voice]}
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

func listItemsForAudio(conn *sql.DB, source string, limit int) ([]itemRow, error) {
	query := `
		SELECT id, source, category, learning_order, source_index, pinyin, hanzi, meaning,
		       sentence_pinyin, sentence_hanzi, sentence_hanzi_raw, sentence_meaning,
		       row_hash, sentence_audio_paths_json
		FROM cram_items`
	args := []any{}
	if strings.TrimSpace(source) != "" {
		query += " WHERE source = ?"
		args = append(args, strings.TrimSpace(source))
	}
	query += " ORDER BY source, learning_order"
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
		item, err := scanItemRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
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
		outPath := filepath.ToSlash(filepath.Join("artifacts", "cram", item.Source, "sentences", item.ID, voice+".ogg"))
		artifactPath, err := gen(item.SentenceHanzi, voice, DefaultVoiceRates[voice], outPath)
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
	if err := tx.QueryRow("SELECT sentence_audio_paths_json FROM cram_items WHERE id = ?", itemID).Scan(&audioJSON); err != nil {
		return err
	}
	audioPaths := map[string]string{}
	_ = json.Unmarshal([]byte(audioJSON), &audioPaths)
	audioPaths[voice] = path
	encoded, err := json.Marshal(audioPaths)
	if err != nil {
		return err
	}
	if _, err := tx.Exec("UPDATE cram_items SET sentence_audio_paths_json = ?, updated_at = ? WHERE id = ?", string(encoded), nowText, itemID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func clearAudioPaths(conn *sql.DB, source string, nowText string) error {
	query := "UPDATE cram_items SET sentence_audio_paths_json = '{}', updated_at = ?"
	args := []any{nowText}
	if strings.TrimSpace(source) != "" {
		query += " WHERE source = ?"
		args = append(args, strings.TrimSpace(source))
	}
	_, err := conn.Exec(query, args...)
	return err
}

func generateSentenceAudio(text, voice, rate, outPath string) (string, error) {
	result, err := audio.TTSAudioWithRate(text, voice, rate, outPath, "edge-tts", "cram_sentence_tts")
	if err != nil {
		return "", err
	}
	if len(result.Artifacts) == 0 {
		return "", fmt.Errorf("tts generated no artifact")
	}
	return result.Artifacts[0].Path, nil
}
