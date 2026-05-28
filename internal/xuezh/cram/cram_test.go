package cram

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func useTestWorkspace(t *testing.T) string {
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
	return workspace
}

func TestTravelImportAndTwoButtonScheduler(t *testing.T) {
	workspace := useTestWorkspace(t)

	result, err := ImportTravelSurvival(ImportOptions{Path: "testdata/travel.txt", AudioMode: "none"})
	if err != nil {
		t.Fatalf("import travel: %v", err)
	}
	if result.RowsSeen != 2 || result.RowsInserted != 2 {
		t.Fatalf("unexpected import result: %+v", result)
	}

	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	overview, err := OverviewFor(now)
	if err != nil {
		t.Fatalf("overview: %v", err)
	}
	if len(overview.Categories) != 1 || overview.Categories[0].TotalCount != 2 {
		t.Fatalf("unexpected overview: %+v", overview)
	}

	cards, err := NextCards(NextOptions{Limit: 1}, now)
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if len(cards) != 1 || cards[0].Source != SourceTravelSurvival || cards[0].Category != "Must Know - Restaurant Vendor Flow" {
		t.Fatalf("unexpected card: %+v", cards)
	}

	grade, err := GradeCard(GradeOptions{ItemID: cards[0].ItemID, Grade: GradeCorrect, SessionID: "s1", Round: 1}, now)
	if err != nil {
		t.Fatalf("grade: %v", err)
	}
	if grade.IntervalMinutes != 8640 || grade.NextDueAt != "2026-05-02T12:00:00+00:00" ||
		grade.Score != 600 || grade.CorrectCount != 1 || grade.ReviewedCount != 1 {
		t.Fatalf("unexpected correct grade: %+v", grade)
	}

	conn, err := sql.Open("sqlite3", filepath.Join(workspace, "db.sqlite3"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer conn.Close()
	var events int
	if err := conn.QueryRow("SELECT COUNT(*) FROM review_events WHERE item_id = ?", cards[0].ItemID).Scan(&events); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if events != 1 {
		t.Fatalf("expected one review event, got %d", events)
	}

	undo, err := UndoLastGrade(UndoGradeOptions{ItemID: cards[0].ItemID, SessionID: "s1"})
	if err != nil {
		t.Fatalf("undo grade: %v", err)
	}
	if undo.UndoneGrade != GradeCorrect || undo.IntervalMinutes != 0 || undo.NextDueAt != nil || undo.Score != nil {
		t.Fatalf("unexpected undo result: %+v", undo)
	}
	if err := conn.QueryRow("SELECT COUNT(*) FROM review_events WHERE item_id = ?", cards[0].ItemID).Scan(&events); err != nil {
		t.Fatalf("count events after undo: %v", err)
	}
	if events != 0 {
		t.Fatalf("expected no review events after undo, got %d", events)
	}
}

func TestApplyPlecoAnswer(t *testing.T) {
	settings := defaultScoringSettings()
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)

	correct := applyPlecoAnswer(scoreState{}, settings, GradeCorrect, now)
	if !correct.Scored || correct.After.Score == nil || *correct.After.Score != 600 ||
		correct.After.Difficulty != 104 || correct.After.History != "6" ||
		correct.After.Correct != 1 || correct.After.Reviewed != 1 {
		t.Fatalf("unexpected first correct update: %+v", correct.After)
	}

	existingScore := 600
	nextCorrect := applyPlecoAnswer(scoreState{Score: &existingScore, Difficulty: 100, Correct: 1, Reviewed: 1}, settings, GradeCorrect, now.Add(25*time.Hour))
	if nextCorrect.After.Score == nil || *nextCorrect.After.Score != 1716 || nextCorrect.After.Difficulty != 104 || nextCorrect.After.Correct != 2 {
		t.Fatalf("unexpected score increase: %+v", nextCorrect.After)
	}

	incorrect := applyPlecoAnswer(scoreState{Score: &existingScore, Difficulty: 100, Correct: 1, Reviewed: 1, History: "6"}, settings, GradeIncorrect, now.Add(25*time.Hour))
	if incorrect.After.Score == nil || *incorrect.After.Score != 100 || incorrect.After.Difficulty != 90 ||
		incorrect.After.History != "62" || incorrect.After.Incorrect != 1 || incorrect.After.Reviewed != 2 {
		t.Fatalf("unexpected incorrect update: %+v", incorrect.After)
	}

	alreadyReviewed := now.Add(-time.Hour)
	oncePerDay := applyPlecoAnswer(scoreState{Score: &existingScore, Difficulty: 100, Correct: 1, Reviewed: 1, History: "6", LastReviewedAt: &alreadyReviewed}, settings, GradeCorrect, now)
	if oncePerDay.Scored || oncePerDay.After.Correct != 1 || oncePerDay.After.History != "6" || *oncePerDay.After.Score != 600 {
		t.Fatalf("score-once-per-day should leave score row unchanged: %+v", oncePerDay.After)
	}
}

func TestReviewSessionPersistsQueueGradeAndUndo(t *testing.T) {
	_ = useTestWorkspace(t)
	if _, err := ImportTravelSurvival(ImportOptions{Path: "testdata/travel.txt", AudioMode: "none"}); err != nil {
		t.Fatalf("import travel: %v", err)
	}
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	session, err := StartReviewSession(ReviewSessionStartOptions{Limit: 2}, now)
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	if session.ID == "" || session.Card == nil || len(session.Queue) != 1 || session.Revealed {
		t.Fatalf("unexpected fresh session: %+v", session)
	}
	if _, _, err := GradeReviewSession(GradeOptions{SessionID: session.ID, ItemID: session.Card.ItemID, Grade: GradeCorrect}, now.Add(time.Second)); err == nil {
		t.Fatalf("grading before reveal should fail")
	}
	session, err = RevealReviewSession(session.ID, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("reveal session: %v", err)
	}
	if !session.Revealed {
		t.Fatalf("expected revealed session")
	}
	firstID := session.Card.ItemID
	session, grade, err := GradeReviewSession(GradeOptions{SessionID: session.ID, ItemID: firstID, Grade: GradeIncorrect}, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("grade session: %v", err)
	}
	if grade.IncorrectCount != 1 || len(session.ReviewedCards) != 1 || len(session.RetryQueue) != 1 || session.Card == nil || session.Card.ItemID == firstID {
		t.Fatalf("incorrect grade should save score and defer retry: grade=%+v session=%+v", grade, session)
	}
	active, err := ActiveReviewSession()
	if err != nil {
		t.Fatalf("active session: %v", err)
	}
	if active == nil || active.ID != session.ID || active.Card == nil || active.Card.ItemID != session.Card.ItemID || len(active.RetryQueue) != 1 {
		t.Fatalf("active session did not persist queue: %+v", active)
	}
	secondID := session.Card.ItemID
	session, err = RevealReviewSession(session.ID, now.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("reveal second: %v", err)
	}
	session, grade, err = GradeReviewSession(GradeOptions{SessionID: session.ID, ItemID: secondID, Grade: GradeCorrect}, now.Add(4*time.Minute))
	if err != nil {
		t.Fatalf("grade second: %v", err)
	}
	if grade.CorrectCount != 1 || session.Card == nil || session.Card.ItemID != firstID || !session.WasRetry || session.Round != 2 {
		t.Fatalf("expected retry card after queue drains: grade=%+v session=%+v", grade, session)
	}
	session, undo, err := UndoReviewSession(session.ID, now.Add(5*time.Minute))
	if err != nil {
		t.Fatalf("undo session: %v", err)
	}
	if undo.UndoneGrade != GradeCorrect || session.Card == nil || session.Card.ItemID != secondID || !session.Revealed || len(session.RetryQueue) != 1 {
		t.Fatalf("undo should restore score and prior queue snapshot: undo=%+v session=%+v", undo, session)
	}
}

func TestDuePreviewAgesWithTime(t *testing.T) {
	_ = useTestWorkspace(t)
	if _, err := ImportTravelSurvival(ImportOptions{Path: "testdata/travel.txt", AudioMode: "none"}); err != nil {
		t.Fatalf("import travel: %v", err)
	}
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	cards, err := NextCards(NextOptions{Limit: 1}, now)
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if _, err := GradeCard(GradeOptions{ItemID: cards[0].ItemID, Grade: GradeCorrect, SessionID: "due"}, now); err != nil {
		t.Fatalf("grade correct: %v", err)
	}
	filters := PracticeFilters{IncludeDue: true, ScoreBelow: 200, IncludeNoScore: true}
	beforeDue, err := PracticePreviewFor(filters, now.Add(5*24*time.Hour))
	if err != nil {
		t.Fatalf("preview before due: %v", err)
	}
	for _, card := range beforeDue.Cards {
		if card.ItemID == cards[0].ItemID {
			t.Fatalf("card should not be due after five days: %+v", card)
		}
	}
	afterDue, err := PracticePreviewFor(filters, now.Add(7*24*time.Hour))
	if err != nil {
		t.Fatalf("preview after due: %v", err)
	}
	found := false
	for _, card := range afterDue.Cards {
		if card.ItemID == cards[0].ItemID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("card should become due as time passes: %+v", afterDue.Cards)
	}
}

func TestHelloChinesePlecoTextImport(t *testing.T) {
	_ = useTestWorkspace(t)

	result, err := ImportHelloChinese(ImportOptions{Path: "testdata/hellochinese.txt", AudioMode: "none"})
	if err != nil {
		t.Fatalf("import hellochinese: %v", err)
	}
	if result.RowsSeen != 3 || result.RowsInserted != 3 {
		t.Fatalf("unexpected import result: %+v", result)
	}

	cards, err := NextCards(NextOptions{Limit: 3}, time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if len(cards) != 3 {
		t.Fatalf("expected three cards, got %d", len(cards))
	}
	if cards[0].Source != SourceHelloChinese || cards[0].Category != "Hello" || cards[0].SentenceHanzi != "你是龙大。" {
		t.Fatalf("unexpected first card: %+v", cards[0])
	}
	if cards[0].SentencePinyin != "nǐ shì lóng dà" {
		t.Fatalf("expected generated sentence pinyin, got %q", cards[0].SentencePinyin)
	}
	if cards[2].SentenceHanzi != "你是中国人吗？" {
		t.Fatalf("expected punctuation-normalized sentence, got %q", cards[2].SentenceHanzi)
	}
}

func TestLearnerStateIsCompactAndStable(t *testing.T) {
	_ = useTestWorkspace(t)

	if _, err := ImportHelloChinese(ImportOptions{Path: "testdata/hellochinese.txt", AudioMode: "none"}); err != nil {
		t.Fatalf("import hellochinese: %v", err)
	}
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	state, err := LearnerStateFor(now)
	if err != nil {
		t.Fatalf("learner state: %v", err)
	}
	if state.GeneratedAt == "" || state.ChangedAt == "" || state.StateHash == "" || state.LearnedScore != 200 {
		t.Fatalf("missing state metadata: %+v", state)
	}
	if len(state.Cards) != 3 {
		t.Fatalf("expected three cards, got %d", len(state.Cards))
	}
	expectedColumns := []string{"category", "hanzi", "meaning", "sentence", "sentence_meaning", "score", "learned", "due", "correct", "incorrect", "reviewed", "first_reviewed", "last_reviewed", "next_due", "history"}
	if len(state.Columns) != len(expectedColumns) {
		t.Fatalf("unexpected columns: %+v", state.Columns)
	}
	for i, column := range expectedColumns {
		if state.Columns[i] != column {
			t.Fatalf("unexpected column %d: got %q want %q", i, state.Columns[i], column)
		}
	}
	first := state.Cards[0]
	if len(first) != len(expectedColumns) ||
		first[0] != "HelloChinese / Hello" ||
		first[1] != "你" ||
		first[3] != "你是龙大。" ||
		first[5] != nil ||
		first[6] != false {
		t.Fatalf("unexpected first learner row: %+v", first)
	}
	for _, row := range state.Cards {
		for _, value := range row {
			if value == "nǐ" || value == "wǒ" || value == "shì" {
				t.Fatalf("learner state should omit pinyin: %+v", row)
			}
		}
	}

	again, err := LearnerStateFor(now)
	if err != nil {
		t.Fatalf("learner state again: %v", err)
	}
	if again.StateHash != state.StateHash {
		t.Fatalf("hash should be stable: first=%s again=%s", state.StateHash, again.StateHash)
	}

	cards, err := NextCards(NextOptions{Limit: 1}, now)
	if err != nil {
		t.Fatalf("next cards: %v", err)
	}
	if _, err := GradeCard(GradeOptions{ItemID: cards[0].ItemID, Grade: GradeCorrect, SessionID: "learner"}, now.Add(time.Minute)); err != nil {
		t.Fatalf("grade card: %v", err)
	}
	changed, err := LearnerStateFor(now.Add(2 * time.Minute))
	if err != nil {
		t.Fatalf("learner state after grade: %v", err)
	}
	if changed.StateHash == state.StateHash || changed.Cards[0][5] == nil || changed.Cards[0][6] != true {
		t.Fatalf("learner hash and facts should update after review: before=%s after=%+v", state.StateHash, changed.Cards[0])
	}
}

func TestImportGeneratesAudioWithCleanSentence(t *testing.T) {
	workspace := useTestWorkspace(t)
	var texts []string
	result, err := ImportHelloChinese(ImportOptions{
		Path:      "testdata/hellochinese.txt",
		AudioMode: "sentence",
		Voices:    []string{"zh-CN-XiaoxiaoNeural"},
		AudioGenerator: func(text, voice, rate, outPath string) (string, error) {
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
	cards, err := NextCards(NextOptions{Limit: 1}, time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if cards[0].SentenceAudioPaths["zh-CN-XiaoxiaoNeural"] == "" {
		t.Fatalf("missing audio path: %+v", cards[0].SentenceAudioPaths)
	}
}

func TestPlecoScoreImportUsesMetadataOnly(t *testing.T) {
	workspace := useTestWorkspace(t)
	if _, err := ImportHelloChinese(ImportOptions{Path: "testdata/hellochinese.txt", AudioMode: "none"}); err != nil {
		t.Fatalf("import hellochinese: %v", err)
	}
	if _, err := ImportTravelSurvival(ImportOptions{Path: "testdata/travel.txt", AudioMode: "none"}); err != nil {
		t.Fatalf("import travel: %v", err)
	}
	plecoPath := filepath.Join(workspace, "pleco.pqb")
	createPlecoScoreFixture(t, plecoPath)

	result, err := ImportPlecoScores(PlecoScoreImportOptions{Path: plecoPath})
	if err != nil {
		t.Fatalf("import pleco scores: %v", err)
	}
	if result.CanonicalRows != 5 || result.SeededRows != 5 || result.UnseededRows != 0 {
		t.Fatalf("unexpected score import result: %+v", result)
	}
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	overview, err := OverviewFor(now)
	if err != nil {
		t.Fatalf("overview: %v", err)
	}
	if overview.Sources[0].LearnedCount != 2 {
		t.Fatalf("expected two score-learned HelloChinese cards, got %+v", overview.Sources)
	}
	defaultFilters := PracticeFilters{
		ScoreBelow:        200,
		MissesMoreThan:    0,
		IncludeNotLearned: true,
		IncludeDue:        true,
		IncludeGotWrong:   false,
		IncludeNoScore:    true,
	}
	preview, err := PracticePreviewFor(defaultFilters, now)
	if err != nil {
		t.Fatalf("practice preview: %v", err)
	}
	if len(preview.Cards) != 2 || preview.Cards[0].Word != "吗" || preview.Cards[1].Word != "菜单" {
		t.Fatalf("unexpected default practice preview: %+v", preview.Cards)
	}
	if preview.Sources[0].PracticeCount != 1 || preview.Sources[0].NotLearnedCount != 1 || preview.Sources[0].GotWrongCount != 1 ||
		preview.Sources[1].PracticeCount != 1 || preview.Sources[1].NotLearnedCount != 1 || preview.Sources[1].GotWrongCount != 1 {
		t.Fatalf("unexpected source practice totals: %+v", preview.Sources)
	}
	gotWrong, err := PracticePreviewFor(PracticeFilters{IncludeGotWrong: true, MissesMoreThan: 0}, now)
	if err != nil {
		t.Fatalf("got-wrong preview: %v", err)
	}
	if len(gotWrong.Cards) != 2 || gotWrong.Cards[0].Word != "吗" || gotWrong.Cards[1].Word != "买单" {
		t.Fatalf("unexpected got-wrong preview: %+v", gotWrong.Cards)
	}
	notLearned, err := PracticePreviewFor(PracticeFilters{ScoreBelow: 200, IncludeNotLearned: true, IncludeNoScore: true}, now)
	if err != nil {
		t.Fatalf("not-learned preview: %v", err)
	}
	if len(notLearned.Cards) != 2 || notLearned.Cards[0].Word != "吗" || notLearned.Cards[1].Word != "菜单" {
		t.Fatalf("unexpected not-learned preview: %+v", notLearned.Cards)
	}
	advanced, err := PracticePreviewFor(PracticeFilters{ScoreBelow: 700, IncludeNotLearned: true, IncludeNoScore: true}, now)
	if err != nil {
		t.Fatalf("advanced preview: %v", err)
	}
	if len(advanced.Cards) != 4 {
		t.Fatalf("expected four cards under score 700 plus no-score, got %+v", advanced.Cards)
	}
	if advanced.Sources[0].NotLearnedCount != 1 || advanced.Sources[1].NotLearnedCount != 1 {
		t.Fatalf("learned progress should not change with advanced score filters: %+v", advanced.Sources)
	}
	selected, err := NextCards(NextOptions{Limit: 2, CardIDs: []string{preview.Cards[1].ItemID, preview.Cards[0].ItemID}}, now)
	if err != nil {
		t.Fatalf("selected next: %v", err)
	}
	if len(selected) != 2 || selected[0].Word != "吗" || selected[1].Word != "菜单" {
		t.Fatalf("explicit card id batch should use practice ordering, got %+v", selected)
	}
	cards, err := NextCards(NextOptions{Limit: 5}, now)
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if len(cards) != 2 {
		t.Fatalf("only not-learned or due cards should be returned; got %+v", cards)
	}
	if cards[0].Word != "吗" || cards[1].Word != "菜单" {
		t.Fatalf("unexpected priority order: %+v", cards)
	}

	localReviewTime := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	updated, err := GradeCard(GradeOptions{ItemID: preview.Cards[0].ItemID, Grade: GradeCorrect, SessionID: "local"}, localReviewTime)
	if err != nil {
		t.Fatalf("local grade: %v", err)
	}
	if updated.Score != 600 || updated.CorrectCount != 1 || updated.IncorrectCount != 1 || updated.ReviewedCount != 2 {
		t.Fatalf("unexpected local score update: %+v", updated)
	}
	if _, err := ImportPlecoScores(PlecoScoreImportOptions{Path: plecoPath}); err != nil {
		t.Fatalf("reimport pleco scores: %v", err)
	}
	afterReimport, err := PracticePreviewFor(PracticeFilters{ScoreBelow: 700, IncludeNotLearned: true, IncludeNoScore: true}, localReviewTime)
	if err != nil {
		t.Fatalf("preview after reimport: %v", err)
	}
	var ma PracticeCard
	for _, card := range afterReimport.Cards {
		if card.Word == "吗" {
			ma = card
			break
		}
	}
	if ma.Score == nil || *ma.Score != 600 || ma.CorrectCount != 1 || ma.IncorrectCount != 1 || ma.ReviewedCount != 2 {
		t.Fatalf("older Pleco reimport should not overwrite local score: %+v", ma)
	}
}

func TestPlecoScoreImportPartiallySeedsMismatchedCategory(t *testing.T) {
	workspace := useTestWorkspace(t)
	if _, err := ImportHelloChinese(ImportOptions{Path: "testdata/hellochinese.txt", AudioMode: "none"}); err != nil {
		t.Fatalf("import hellochinese: %v", err)
	}
	plecoPath := filepath.Join(workspace, "pleco-mismatch.pqb")
	createPlecoMismatchFixture(t, plecoPath)

	result, err := ImportPlecoScores(PlecoScoreImportOptions{Path: plecoPath})
	if err != nil {
		t.Fatalf("import pleco scores: %v", err)
	}
	if result.CanonicalRows != 3 || result.SeededRows != 2 || result.UnseededRows != 1 {
		t.Fatalf("unexpected score import result: %+v", result)
	}
	preview, err := PracticePreviewFor(PracticeFilters{IncludeNotLearned: true, IncludeNoScore: true, ScoreBelow: 200}, time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("practice preview: %v", err)
	}
	if len(preview.Cards) != 2 || preview.Cards[0].Word != "吗" || preview.Cards[1].Word != "我" {
		t.Fatalf("expected one unseeded row plus one low-score row, got %+v", preview.Cards)
	}
}

func TestOfflineDeckAndIdempotentSync(t *testing.T) {
	workspace := useTestWorkspace(t)
	if _, err := ImportTravelSurvival(ImportOptions{
		Path:      "testdata/travel.txt",
		AudioMode: "sentence",
		Voices:    []string{"zh-CN-XiaoyiNeural"},
		AudioGenerator: func(text, voice, rate, outPath string) (string, error) {
			fullPath := filepath.Join(workspace, outPath)
			if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
				return "", err
			}
			if err := os.WriteFile(fullPath, []byte("ogg"), 0o644); err != nil {
				return "", err
			}
			return outPath, nil
		},
	}); err != nil {
		t.Fatalf("import travel: %v", err)
	}
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	deck, err := OfflineDeck(now)
	if err != nil {
		t.Fatalf("offline deck: %v", err)
	}
	if len(deck.Cards) != 2 || len(deck.AudioPaths) != 2 || deck.Settings.LearnedScore != 200 || deck.Settings.PointsPerDay != 100 {
		t.Fatalf("unexpected offline deck: %+v", deck)
	}

	events := []OfflineReviewEvent{
		{
			EventID:    "offline-2",
			SessionID:  "offline-session",
			ItemID:     deck.Cards[1].ItemID,
			Grade:      GradeCorrect,
			AnsweredAt: "2026-04-27T12:00:00Z",
			Round:      1,
		},
		{
			EventID:    "offline-1",
			SessionID:  "offline-session",
			ItemID:     deck.Cards[0].ItemID,
			Grade:      GradeIncorrect,
			AnsweredAt: "2026-04-26T12:00:00Z",
			Round:      1,
		},
	}
	result, err := SyncOfflineReviewEvents(events, now)
	if err != nil {
		t.Fatalf("sync offline events: %v", err)
	}
	if result.Applied != 2 || result.Skipped != 0 {
		t.Fatalf("unexpected first sync result: %+v", result)
	}
	again, err := SyncOfflineReviewEvents(events, now)
	if err != nil {
		t.Fatalf("sync duplicate offline events: %v", err)
	}
	if again.Applied != 0 || again.Skipped != 2 {
		t.Fatalf("duplicate sync should be idempotent: %+v", again)
	}

	conn, err := sql.Open("sqlite3", filepath.Join(workspace, "db.sqlite3"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer conn.Close()
	var eventsStored int
	if err := conn.QueryRow("SELECT COUNT(*) FROM review_events WHERE session_id = ?", "offline-session").Scan(&eventsStored); err != nil {
		t.Fatalf("count offline events: %v", err)
	}
	if eventsStored != 2 {
		t.Fatalf("expected exactly two stored offline events, got %d", eventsStored)
	}
	refreshed, err := OfflineDeck(now)
	if err != nil {
		t.Fatalf("refreshed offline deck: %v", err)
	}
	if refreshed.Cards[0].IncorrectCount != 1 || refreshed.Cards[1].CorrectCount != 1 || refreshed.Cards[1].Score == nil || *refreshed.Cards[1].Score != 600 {
		t.Fatalf("offline sync should update live score rows once: %+v", refreshed.Cards)
	}
}

func createPlecoScoreFixture(t *testing.T, path string) {
	t.Helper()
	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer conn.Close()
	statements := []string{
		`CREATE TABLE pleco_flash_cards (id INTEGER PRIMARY KEY, hw TEXT, defn TEXT)`,
		`CREATE TABLE pleco_flash_categories (id INTEGER PRIMARY KEY, name TEXT, parent INTEGER, sort INTEGER)`,
		`CREATE TABLE pleco_flash_categoryassigns (card INTEGER, cat INTEGER, id INTEGER PRIMARY KEY AUTOINCREMENT)`,
		`CREATE TABLE pleco_flash_profiles (id INTEGER PRIMARY KEY, name TEXT, sort INTEGER)`,
		`CREATE TABLE pleco_flash_profilesettings (propset INTEGER, propid TEXT, propvalue TEXT, propisstring INTEGER)`,
		`CREATE TABLE pleco_flash_scores_1 (card INTEGER PRIMARY KEY, score INTEGER, difficulty INTEGER, history TEXT, correct INTEGER, incorrect INTEGER, reviewed INTEGER, sincelastchange INTEGER, firstreviewedtime INTEGER, lastreviewedtime INTEGER, scoreinctime INTEGER, scoredectime INTEGER)`,
		`INSERT INTO pleco_flash_cards (id, hw, defn) VALUES
		  (101, '你', 'you (singular)你是龙大。You are Long Da.'),
		  (102, '我', 'I; me我是龙大。I am Long Da.'),
		  (103, '吗', 'question particle你是中国人吗？Are you Chinese?'),
		  (201, '买@单', 'check please我们买单。We will pay the bill.'),
		  (202, '菜单', 'menu我看菜单。I look at the menu.')`,
		`INSERT INTO pleco_flash_categories (id, name, parent, sort) VALUES (1, 'HelloChinese', -2, 1), (2, 'Hello', 1, 1), (3, 'Travel Survival', -2, 2), (4, 'Must Know - Restaurant Vendor Flow', 3, 1)`,
		`INSERT INTO pleco_flash_categoryassigns (card, cat) VALUES (101, 2), (102, 2), (103, 2), (201, 4), (202, 4)`,
		`INSERT INTO pleco_flash_profiles (id, name, sort) VALUES (2, 'Spaced Repetition', 2)`,
		`INSERT INTO pleco_flash_profilesettings (propset, propid, propvalue, propisstring) VALUES
		  (2, 'pro_cardpointsday', '100', 0),
		  (2, 'pro_learnedselcount', '200', 0),
		  (2, 'pro_scoreautomax', '51200', 0),
		  (2, 'pro_scoreautomin', '100', 0),
		  (2, 'pro_scoredecreaseamt', '100', 0),
		  (2, 'pro_scorediffchange2', '-10', 0),
		  (2, 'pro_scorediffchange6', '4', 0),
		  (2, 'pro_scorediffdivisor', '40', 0),
		  (2, 'pro_scoreinitinterval6', '600', 0),
		  (2, 'pro_scoreintervalmult6', '110', 0),
		  (2, 'pro_scoremaxdifficulty', '200', 0),
		  (2, 'pro_scoremindifficulty', '50', 0),
		  (2, 'pro_scoreonceaday', '1', 0)`,
		`INSERT INTO pleco_flash_scores_1 (card, score, difficulty, history, correct, incorrect, reviewed, firstreviewedtime, lastreviewedtime) VALUES (101, 21842, 100, '666666', 6, 0, 6, 1790000000, 1790000000)`,
		`INSERT INTO pleco_flash_scores_1 (card, score, difficulty, history, correct, incorrect, reviewed, firstreviewedtime, lastreviewedtime) VALUES (102, 600, 100, '6', 1, 0, 1, 1790000000, 1790000000)`,
		`INSERT INTO pleco_flash_scores_1 (card, score, difficulty, history, correct, incorrect, reviewed, firstreviewedtime, lastreviewedtime) VALUES (103, 100, 100, '2', 0, 1, 1, 1790000000, 1790000000)`,
		`INSERT INTO pleco_flash_scores_1 (card, score, difficulty, history, correct, incorrect, reviewed, firstreviewedtime, lastreviewedtime) VALUES (201, 600, 100, '62', 1, 1, 2, 1790000000, 1790000000)`,
	}
	for _, statement := range statements {
		if _, err := conn.Exec(statement); err != nil {
			t.Fatalf("fixture statement failed: %v", err)
		}
	}
}

func createPlecoMismatchFixture(t *testing.T, path string) {
	t.Helper()
	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer conn.Close()
	statements := []string{
		`CREATE TABLE pleco_flash_cards (id INTEGER PRIMARY KEY, hw TEXT, defn TEXT)`,
		`CREATE TABLE pleco_flash_categories (id INTEGER PRIMARY KEY, name TEXT, parent INTEGER, sort INTEGER)`,
		`CREATE TABLE pleco_flash_categoryassigns (card INTEGER, cat INTEGER, id INTEGER PRIMARY KEY AUTOINCREMENT)`,
		`CREATE TABLE pleco_flash_profiles (id INTEGER PRIMARY KEY, name TEXT, sort INTEGER)`,
		`CREATE TABLE pleco_flash_profilesettings (propset INTEGER, propid TEXT, propvalue TEXT, propisstring INTEGER)`,
		`CREATE TABLE pleco_flash_scores_1 (card INTEGER PRIMARY KEY, score INTEGER, difficulty INTEGER, history TEXT, correct INTEGER, incorrect INTEGER, reviewed INTEGER, sincelastchange INTEGER, firstreviewedtime INTEGER, lastreviewedtime INTEGER, scoreinctime INTEGER, scoredectime INTEGER)`,
		`INSERT INTO pleco_flash_cards (id, hw, defn) VALUES
		  (101, '你', 'you (singular)你是龙大。You are Long Da.'),
		  (103, '吗', 'question particle你是中国人吗？Are you Chinese?')`,
		`INSERT INTO pleco_flash_categories (id, name, parent, sort) VALUES (1, 'HelloChinese', -2, 1), (2, 'Hello', 1, 1)`,
		`INSERT INTO pleco_flash_categoryassigns (card, cat) VALUES (101, 2), (103, 2)`,
		`INSERT INTO pleco_flash_profiles (id, name, sort) VALUES (2, 'Spaced Repetition', 2)`,
		`INSERT INTO pleco_flash_scores_1 (card, score, difficulty, history, correct, incorrect, reviewed, firstreviewedtime, lastreviewedtime) VALUES (101, 600, 100, '6', 1, 0, 1, 1790000000, 1790000000)`,
		`INSERT INTO pleco_flash_scores_1 (card, score, difficulty, history, correct, incorrect, reviewed, firstreviewedtime, lastreviewedtime) VALUES (103, 100, 100, '2', 0, 1, 1, 1790000000, 1790000000)`,
	}
	for _, statement := range statements {
		if _, err := conn.Exec(statement); err != nil {
			t.Fatalf("fixture statement failed: %v", err)
		}
	}
}
