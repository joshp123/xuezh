package service

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joshp123/xuezh/internal/xuezh/clock"
	"github.com/joshp123/xuezh/internal/xuezh/cram"
	"github.com/joshp123/xuezh/internal/xuezh/db"
	"github.com/joshp123/xuezh/internal/xuezh/ids"
	"github.com/joshp123/xuezh/internal/xuezh/srs"
)

func TestAppLearnerStateMatchesCurrentCramState(t *testing.T) {
	useServiceTestWorkspace(t)
	if _, err := cram.ImportHelloChinese(cram.ImportOptions{Path: filepath.Join("..", "cram", "testdata", "hellochinese.txt"), AudioMode: "none"}); err != nil {
		t.Fatalf("import hellochinese: %v", err)
	}

	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	state, err := New().LearnerState(now)
	if err != nil {
		t.Fatal(err)
	}
	if state.StateHash == "" || len(state.Cards) != 3 {
		t.Fatalf("unexpected learner state: %+v", state)
	}
}

func TestAppSnapshotReturnsFactualReportPayload(t *testing.T) {
	useServiceTestWorkspace(t)

	result, err := New().Snapshot("7d", 5, 10, 64*1024)
	if err != nil {
		t.Fatal(err)
	}
	if result.Data["window"] != "7d" || result.Truncated {
		t.Fatalf("unexpected snapshot result: %+v", result)
	}
}

func TestAppStartReviewReturnsDueRecallAndPronunciationItems(t *testing.T) {
	useServiceTestWorkspace(t)
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	itemID := ids.WordID("吗", "ma")
	recallDue := clock.FormatISO(now.Add(-time.Hour))
	pronunciationDue := clock.FormatISO(now.Add(-2 * time.Hour))
	recallGrade := 3
	pronunciationGrade := 4
	if err := srs.UpsertKnowledge(itemID, &recallDue, &recallGrade, &pronunciationDue, &pronunciationGrade, now.Add(-24*time.Hour)); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}

	result, err := New().StartReview(10, now)
	if err != nil {
		t.Fatal(err)
	}
	if result.GeneratedAt != clock.FormatISO(now) {
		t.Fatalf("generated_at = %q", result.GeneratedAt)
	}
	if len(result.RecallItems) != 1 || result.RecallItems[0].ItemID != itemID {
		t.Fatalf("unexpected recall items: %+v", result.RecallItems)
	}
	if len(result.PronunciationItems) != 1 || result.PronunciationItems[0].ItemID != itemID {
		t.Fatalf("unexpected pronunciation items: %+v", result.PronunciationItems)
	}
}

func TestAppGradeReviewRecordsKnowledgeAndEvents(t *testing.T) {
	useServiceTestWorkspace(t)
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	itemID := ids.WordID("茶", "cha")
	recallGrade := 4
	pronunciationGrade := 3

	result, err := New().GradeReview(GradeReviewOptions{
		ItemID:             itemID,
		RecallGrade:        &recallGrade,
		PronunciationGrade: &pronunciationGrade,
		Rule:               "leitner",
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if result.ItemID != itemID || result.RecallGrade == nil || *result.RecallGrade != recallGrade {
		t.Fatalf("unexpected grade result: %+v", result)
	}
	if result.PronunciationGrade == nil || *result.PronunciationGrade != pronunciationGrade {
		t.Fatalf("unexpected pronunciation result: %+v", result)
	}
	if countReviewEvents(t, itemID, "review.grade") != 2 {
		t.Fatalf("grade should record one event per reviewed vector")
	}
}

func TestAppGradeReviewRollsBackKnowledgeWhenEventInsertFails(t *testing.T) {
	useServiceTestWorkspace(t)
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	itemID := ids.WordID("书", "shu")
	recallGrade := 4
	conn := openServiceTestDB(t)
	defer conn.Close()
	if _, err := conn.Exec(`
		CREATE TRIGGER fail_review_events
		BEFORE INSERT ON review_events
		BEGIN
			SELECT RAISE(ABORT, 'forced event failure');
		END;
	`); err != nil {
		t.Fatal(err)
	}

	_, err := New().GradeReview(GradeReviewOptions{ItemID: itemID, RecallGrade: &recallGrade}, now)
	if err == nil {
		t.Fatal("expected grade review to fail")
	}
	if countKnowledgeRows(t, itemID) != 0 {
		t.Fatal("knowledge row survived failed review event insert")
	}
}

func TestAppBuryReviewDefersRecallAndRecordsEvent(t *testing.T) {
	useServiceTestWorkspace(t)
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	itemID := ids.WordID("饭", "fan")

	result, err := New().BuryReview(itemID, "too_easy", now)
	if err != nil {
		t.Fatal(err)
	}
	if result.ItemID != itemID || result.Reason != "too_easy" || result.NextDue == "" {
		t.Fatalf("unexpected bury result: %+v", result)
	}
	if countReviewEvents(t, itemID, "review.bury") != 1 {
		t.Fatalf("bury should record one event")
	}
}

func TestAppPreviewSRSReturnsRecallAndPronunciationForecasts(t *testing.T) {
	useServiceTestWorkspace(t)
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	itemID := ids.WordID("水", "shui")
	recallDue := clock.FormatISO(now.Add(48 * time.Hour))
	pronunciationDue := clock.FormatISO(now.Add(72 * time.Hour))
	recallGrade := 4
	pronunciationGrade := 3
	if err := srs.UpsertKnowledge(itemID, &recallDue, &recallGrade, &pronunciationDue, &pronunciationGrade, now); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}

	result, err := New().PreviewSRS(7, now)
	if err != nil {
		t.Fatal(err)
	}
	if result.Days != 7 {
		t.Fatalf("days = %d", result.Days)
	}
	if result.Forecast["recall"]["2026-04-28"] != 1 {
		t.Fatalf("unexpected recall forecast: %+v", result.Forecast["recall"])
	}
	if result.Forecast["pronunciation"]["2026-04-29"] != 1 {
		t.Fatalf("unexpected pronunciation forecast: %+v", result.Forecast["pronunciation"])
	}
}

func useServiceTestWorkspace(t *testing.T) {
	t.Helper()
	workspace := t.TempDir()
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	configPath := filepath.Join(configHome, "xuezh", "config.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "[workspace]\ndir = \"" + workspace + "\"\n"
	if err := os.WriteFile(configPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func countReviewEvents(t *testing.T, itemID, eventType string) int {
	t.Helper()
	conn := openServiceTestDB(t)
	defer conn.Close()
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM review_events WHERE item_id = ? AND event_type = ?", itemID, eventType).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func countKnowledgeRows(t *testing.T, itemID string) int {
	t.Helper()
	conn := openServiceTestDB(t)
	defer conn.Close()
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM user_knowledge WHERE item_id = ?", itemID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func openServiceTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath, err := db.InitDB()
	if err != nil {
		t.Fatal(err)
	}
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	return conn
}
