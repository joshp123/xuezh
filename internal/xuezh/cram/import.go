package cram

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/joshp123/xuezh/internal/xuezh/clock"
)

func ImportHelloChinese(opts ImportOptions) (ImportResult, error) {
	return importRows(opts, newPlecoTextParser(SourceHelloChinese, "HelloChinese"))
}

func ImportTravelSurvival(opts ImportOptions) (ImportResult, error) {
	return importRows(opts, newPlecoTextParser(SourceTravelSurvival, "Travel Survival"))
}

func importRows(opts ImportOptions, parse func([]byte, int, *string, *int) ([]itemRow, error)) (ImportResult, error) {
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
	var currentCategory string
	entryIndex := 0
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		rows, err := parse(line, result.RowsSeen+1, &currentCategory, &entryIndex)
		if err != nil {
			return result, err
		}
		if len(rows) == 0 {
			continue
		}
		for _, row := range rows {
			result.RowsSeen++
			item, inserted, err := upsertItem(conn, row, nowText)
			if err != nil {
				return result, err
			}
			if inserted {
				result.RowsInserted++
			} else {
				result.RowsExisting++
			}
			if audioMode == "sentence" {
				audioResult, err := ensureAudio(conn, item, voices, gen, nowText)
				result.AudioGenerated += audioResult.generated
				result.AudioExisting += audioResult.existing
				if err != nil {
					result.AudioFailed++
					return result, err
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return result, err
	}
	return result, nil
}

func newPlecoTextParser(source, root string) func([]byte, int, *string, *int) ([]itemRow, error) {
	return func(line []byte, lineNumber int, currentCategory *string, entryIndex *int) ([]itemRow, error) {
		return parsePlecoTextRow(source, root, line, lineNumber, currentCategory, entryIndex)
	}
}

func parsePlecoTextRow(source, root string, line []byte, lineNumber int, currentCategory *string, entryIndex *int) ([]itemRow, error) {
	raw := strings.TrimSpace(strings.TrimPrefix(string(line), "\ufeff"))
	if raw == "" {
		return nil, nil
	}
	if strings.HasPrefix(raw, "//") {
		category := strings.TrimSpace(strings.TrimPrefix(raw, "//"))
		prefix := root + "/"
		if strings.HasPrefix(category, prefix) {
			category = strings.TrimSpace(strings.TrimPrefix(category, prefix))
		}
		if category == "" {
			category = root
		}
		*currentCategory = category
		return nil, nil
	}
	if *currentCategory == "" {
		return nil, fmt.Errorf("line %d: category header required before row", lineNumber)
	}
	parts := strings.Split(raw, "\t")
	if len(parts) < 3 {
		return nil, fmt.Errorf("line %d: expected tab-delimited hanzi, pinyin, payload", lineNumber)
	}
	fields := strings.Split(parts[2], "\ueab1")
	if len(fields) < 4 {
		return nil, fmt.Errorf("line %d: expected Pleco payload with meaning and sentence", lineNumber)
	}
	*entryIndex = *entryIndex + 1
	order := *entryIndex
	sentence := cleanChineseSentence(fields[2])
	if strings.TrimSpace(parts[0]) == "" || sentence == "" {
		return nil, fmt.Errorf("line %d: hanzi and sentence_hanzi are required", lineNumber)
	}
	row := itemRow{
		Source:           source,
		Category:         strings.TrimSpace(*currentCategory),
		LearningOrder:    order,
		SourceIndex:      order,
		Pinyin:           strings.TrimSpace(parts[1]),
		Hanzi:            strings.TrimSpace(parts[0]),
		Meaning:          strings.TrimSpace(fields[0]),
		SentencePinyin:   sentencePinyin(sentence),
		SentenceHanzi:    sentence,
		SentenceHanziRaw: strings.TrimSpace(fields[2]),
		SentenceMeaning:  strings.TrimSpace(fields[3]),
	}
	row.RowHash = hashRow(row)
	return []itemRow{row}, nil
}
