package hellochinese

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestImportNextAndGrade(t *testing.T) {
	workspace := t.TempDir()
	t.Setenv("XUEZH_WORKSPACE_DIR", workspace)

	result, err := ImportCorpus(ImportOptions{
		Path:      "testdata/min.jsonl",
		AudioMode: "none",
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if result.RowsSeen != 3 || result.RowsInserted != 3 {
		t.Fatalf("unexpected import result: %+v", result)
	}

	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	cards, err := NextCards(1, now)
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if len(cards) != 1 {
		t.Fatalf("expected one card, got %d", len(cards))
	}
	if cards[0].LearningOrder != 1 || cards[0].Word != "你" || cards[0].SentenceHanzi != "你是龙大。" {
		t.Fatalf("unexpected first card: %+v", cards[0])
	}

	grade, err := GradeCard(cards[0].ItemID, "good", now)
	if err != nil {
		t.Fatalf("grade: %v", err)
	}
	if grade.NextDueAt != "2026-04-26T14:00:00+00:00" || grade.SeenCount != 1 {
		t.Fatalf("unexpected grade result: %+v", grade)
	}

	cards, err = NextCards(1, now)
	if err != nil {
		t.Fatalf("next after grade: %v", err)
	}
	if len(cards) != 1 || cards[0].LearningOrder != 2 {
		t.Fatalf("expected second card after good grade, got %+v", cards)
	}
}

func TestImportIsIdempotentAndDetectsConflict(t *testing.T) {
	workspace := t.TempDir()
	t.Setenv("XUEZH_WORKSPACE_DIR", workspace)
	if _, err := ImportCorpus(ImportOptions{Path: "testdata/min.jsonl", AudioMode: "none"}); err != nil {
		t.Fatalf("first import: %v", err)
	}
	result, err := ImportCorpus(ImportOptions{Path: "testdata/min.jsonl", AudioMode: "none"})
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if result.RowsInserted != 0 || result.RowsExisting != 3 {
		t.Fatalf("expected idempotent import, got %+v", result)
	}

	conflictPath := filepath.Join(t.TempDir(), "conflict.jsonl")
	body := `{"index":1,"unit_label":"Basics - 1/4","pinyin":"tā","hanzi":"他","meaning":"he","sentence_pinyin":"tā shì lóngdà","sentence_hanzi":"他 是 龙大。","sentence_meaning":"He is Long Da."}` + "\n"
	if err := os.WriteFile(conflictPath, []byte(body), 0o644); err != nil {
		t.Fatalf("write conflict fixture: %v", err)
	}
	if _, err := ImportCorpus(ImportOptions{Path: conflictPath, AudioMode: "none"}); err == nil {
		t.Fatal("expected conflict error")
	}
}

func TestImportGeneratesAudioWithCleanSentence(t *testing.T) {
	workspace := t.TempDir()
	t.Setenv("XUEZH_WORKSPACE_DIR", workspace)
	var texts []string
	result, err := ImportCorpus(ImportOptions{
		Path:      "testdata/min.jsonl",
		AudioMode: "sentence",
		Voices:    []string{"zh-CN-XiaoxiaoNeural"},
		AudioGenerator: func(text, voice, outPath string) (string, error) {
			texts = append(texts, text)
			fullPath := filepath.Join(workspace, outPath)
			if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
				return "", err
			}
			if err := os.WriteFile(fullPath, []byte("ogg"), 0o644); err != nil {
				return "", err
			}
			return outPath, nil
		},
	})
	if err != nil {
		t.Fatalf("import with audio: %v", err)
	}
	if result.AudioGenerated != 3 {
		t.Fatalf("expected three audio files, got %+v", result)
	}
	if texts[0] != "你是龙大。" {
		t.Fatalf("expected cleaned sentence text, got %q", texts[0])
	}
	cards, err := NextCards(1, time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if cards[0].SentenceAudioPaths["zh-CN-XiaoxiaoNeural"] == "" {
		t.Fatalf("missing audio path: %+v", cards[0].SentenceAudioPaths)
	}
}

func TestBackfillAudioPreservesAllVoicesWithConcurrentWorkers(t *testing.T) {
	workspace := t.TempDir()
	t.Setenv("XUEZH_WORKSPACE_DIR", workspace)
	if _, err := ImportCorpus(ImportOptions{Path: "testdata/min.jsonl", AudioMode: "none"}); err != nil {
		t.Fatalf("import: %v", err)
	}

	voices := []string{"v1", "v2", "v3", "v4"}
	result, err := BackfillAudio(AudioBackfillOptions{
		Voices:      voices,
		Concurrency: 4,
		AudioGenerator: func(text, voice, outPath string) (string, error) {
			fullPath := filepath.Join(workspace, outPath)
			if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
				return "", err
			}
			if err := os.WriteFile(fullPath, []byte(text+" "+voice), 0o644); err != nil {
				return "", err
			}
			return outPath, nil
		},
	})
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if result.TasksSeen != 12 || result.AudioGenerated != 12 || result.AudioFailed != 0 {
		t.Fatalf("unexpected backfill result: %+v", result)
	}

	cards, err := NextCards(3, time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	for _, card := range cards {
		if len(card.SentenceAudioPaths) != len(voices) {
			t.Fatalf("card %d lost voice paths: %+v", card.LearningOrder, card.SentenceAudioPaths)
		}
		for _, voice := range voices {
			if card.SentenceAudioPaths[voice] == "" {
				t.Fatalf("card %d missing %s: %+v", card.LearningOrder, voice, card.SentenceAudioPaths)
			}
		}
	}
}
