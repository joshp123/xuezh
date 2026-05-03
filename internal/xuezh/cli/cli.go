package cli

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joshp123/xuezh/internal/xuezh/audio"
	"github.com/joshp123/xuezh/internal/xuezh/clock"
	"github.com/joshp123/xuezh/internal/xuezh/config"
	"github.com/joshp123/xuezh/internal/xuezh/content"
	"github.com/joshp123/xuezh/internal/xuezh/cram"
	"github.com/joshp123/xuezh/internal/xuezh/datasets"
	"github.com/joshp123/xuezh/internal/xuezh/db"
	"github.com/joshp123/xuezh/internal/xuezh/envelope"
	"github.com/joshp123/xuezh/internal/xuezh/events"
	"github.com/joshp123/xuezh/internal/xuezh/jsonio"
	"github.com/joshp123/xuezh/internal/xuezh/paths"
	"github.com/joshp123/xuezh/internal/xuezh/process"
	"github.com/joshp123/xuezh/internal/xuezh/reports"
	"github.com/joshp123/xuezh/internal/xuezh/retention"
	"github.com/joshp123/xuezh/internal/xuezh/snapshot"
	"github.com/joshp123/xuezh/internal/xuezh/srs"
	"github.com/joshp123/xuezh/internal/xuezh/webserver"
)

const version = "0.1.0"

func Run(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 1
	}
	switch args[0] {
	case "version":
		return runVersion(args[1:])
	case "snapshot":
		return runSnapshot(args[1:])
	case "db":
		return runDB(args[1:])
	case "dataset":
		return runDataset(args[1:])
	case "hellochinese":
		return runHelloChinese(args[1:])
	case "travel":
		return runTravel(args[1:])
	case "pleco":
		return runPleco(args[1:])
	case "review":
		return runReview(args[1:])
	case "cram":
		return runCram(args[1:])
	case "srs":
		return runSRS(args[1:])
	case "report":
		return runReport(args[1:])
	case "event":
		return runEvent(args[1:])
	case "content":
		return runContent(args[1:])
	case "audio":
		return runAudio(args[1:])
	case "web":
		return runWeb(args[1:])
	case "doctor":
		return runDoctor(args[1:])
	case "gc":
		return runGC(args[1:])
	default:
		printUsage()
		return 1
	}
}

func runWeb(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 1
	}
	switch args[0] {
	case "serve":
		return runWebServe(args[1:])
	default:
		printUsage()
		return 1
	}
}

func runWebServe(args []string) int {
	fs := flag.NewFlagSet("web serve", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	port := fs.Int("port", 8765, "port")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if err := webserver.Serve(webserver.ServerOptions{Port: *port}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func runHelloChinese(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 1
	}
	switch args[0] {
	case "import":
		return runHelloChineseImport(args[1:])
	case "audio-backfill":
		return runHelloChineseAudioBackfill(args[1:])
	default:
		printUsage()
		return 1
	}
}

func runHelloChineseAudioBackfill(args []string) int {
	fs := flag.NewFlagSet("hellochinese audio-backfill", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	voicesRaw := fs.String("voices", strings.Join(cram.DefaultVoices, ","), "comma-separated voices")
	concurrency := fs.Int("concurrency", 4, "parallel audio workers")
	limit := fs.Int("limit", 0, "max imported items to backfill; 0 means all")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	result, err := cram.BackfillAudio(cram.AudioBackfillOptions{
		Source:      cram.SourceHelloChinese,
		Voices:      splitCSV(*voicesRaw),
		Concurrency: *concurrency,
		Limit:       *limit,
	})
	if err != nil {
		var toolMissing process.ToolMissingError
		if errors.As(err, &toolMissing) {
			return emitTypedError("hellochinese.audio-backfill", "TOOL_MISSING", err.Error(), map[string]any{"tool": toolMissing.Tool})
		}
		var processFailed process.ProcessFailedError
		if errors.As(err, &processFailed) {
			return emitTypedError("hellochinese.audio-backfill", "BACKEND_FAILED", "audio backend failed during HelloChinese backfill", map[string]any{
				"cmd":        processFailed.Cmd,
				"returncode": processFailed.ReturnCode,
				"stderr":     trim(processFailed.Stderr),
			})
		}
		return emitTypedError("hellochinese.audio-backfill", "BACKEND_FAILED", err.Error(), nil)
	}
	out := envelope.OK("hellochinese.audio-backfill", map[string]any{
		"tasks_seen":      result.TasksSeen,
		"audio_generated": result.AudioGenerated,
		"audio_existing":  result.AudioExisting,
		"audio_failed":    result.AudioFailed,
	}, nil, false, map[string]any{"concurrency": *concurrency, "limit": *limit})
	return emit(out)
}

func runHelloChineseImport(args []string) int {
	fs := flag.NewFlagSet("hellochinese import", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	path := fs.String("path", "", "HelloChinese Pleco text import path")
	audioMode := fs.String("audio", "none", "none|sentence")
	voicesRaw := fs.String("voices", strings.Join(cram.DefaultVoices, ","), "comma-separated voices")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	voices := splitCSV(*voicesRaw)
	result, err := cram.ImportHelloChinese(cram.ImportOptions{Path: *path, AudioMode: *audioMode, Voices: voices})
	if err != nil {
		var toolMissing process.ToolMissingError
		if errors.As(err, &toolMissing) {
			return emitTypedError("hellochinese.import", "TOOL_MISSING", err.Error(), map[string]any{"tool": toolMissing.Tool, "path": *path, "audio": *audioMode})
		}
		var processFailed process.ProcessFailedError
		if errors.As(err, &processFailed) {
			return emitTypedError("hellochinese.import", "BACKEND_FAILED", "audio backend failed during HelloChinese import", map[string]any{
				"cmd":        processFailed.Cmd,
				"returncode": processFailed.ReturnCode,
				"stderr":     trim(processFailed.Stderr),
				"path":       *path,
				"audio":      *audioMode,
			})
		}
		return emitTypedError("hellochinese.import", "INVALID_ARGUMENT", err.Error(), map[string]any{"path": *path, "audio": *audioMode})
	}
	out := envelope.OK("hellochinese.import", map[string]any{
		"rows_seen":       result.RowsSeen,
		"rows_inserted":   result.RowsInserted,
		"rows_existing":   result.RowsExisting,
		"audio_generated": result.AudioGenerated,
		"audio_existing":  result.AudioExisting,
		"audio_failed":    result.AudioFailed,
	}, nil, false, nil)
	return emit(out)
}

func runTravel(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 1
	}
	switch args[0] {
	case "import":
		return runTravelImport(args[1:])
	default:
		printUsage()
		return 1
	}
}

func runTravelImport(args []string) int {
	fs := flag.NewFlagSet("travel import", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	path := fs.String("path", "", "Travel Survival Pleco import path")
	audioMode := fs.String("audio", "none", "none|sentence")
	voicesRaw := fs.String("voices", strings.Join(cram.DefaultVoices, ","), "comma-separated voices")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	result, err := cram.ImportTravelSurvival(cram.ImportOptions{Path: *path, AudioMode: *audioMode, Voices: splitCSV(*voicesRaw)})
	if err != nil {
		var toolMissing process.ToolMissingError
		if errors.As(err, &toolMissing) {
			return emitTypedError("travel.import", "TOOL_MISSING", err.Error(), map[string]any{"tool": toolMissing.Tool, "path": *path, "audio": *audioMode})
		}
		var processFailed process.ProcessFailedError
		if errors.As(err, &processFailed) {
			return emitTypedError("travel.import", "BACKEND_FAILED", "audio backend failed during Travel import", map[string]any{
				"cmd":        processFailed.Cmd,
				"returncode": processFailed.ReturnCode,
				"stderr":     trim(processFailed.Stderr),
				"path":       *path,
				"audio":      *audioMode,
			})
		}
		return emitTypedError("travel.import", "INVALID_ARGUMENT", err.Error(), map[string]any{"path": *path, "audio": *audioMode})
	}
	out := envelope.OK("travel.import", map[string]any{
		"rows_seen":       result.RowsSeen,
		"rows_inserted":   result.RowsInserted,
		"rows_existing":   result.RowsExisting,
		"audio_generated": result.AudioGenerated,
		"audio_existing":  result.AudioExisting,
		"audio_failed":    result.AudioFailed,
	}, nil, false, nil)
	return emit(out)
}

func runPleco(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 1
	}
	switch args[0] {
	case "score-import":
		return runPlecoScoreImport(args[1:])
	default:
		printUsage()
		return 1
	}
}

func runPlecoScoreImport(args []string) int {
	fs := flag.NewFlagSet("pleco score-import", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	path := fs.String("path", "", "Pleco backup .pqb path")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	result, err := cram.ImportPlecoScores(cram.PlecoScoreImportOptions{Path: *path})
	if err != nil {
		return emitTypedError("pleco.score-import", "INVALID_ARGUMENT", err.Error(), map[string]any{"path": *path})
	}
	out := envelope.OK("pleco.score-import", map[string]any{
		"canonical_rows":      result.CanonicalRows,
		"seeded_rows":         result.SeededRows,
		"unseeded_rows":       result.UnseededRows,
		"unseeded_categories": result.UnseededCategories,
	}, nil, false, nil)
	return emit(out)
}

func runCram(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 1
	}
	switch args[0] {
	case "overview":
		return runCramOverview(args[1:])
	case "audio-backfill":
		return runCramAudioBackfill(args[1:])
	case "next":
		return runCramNext(args[1:])
	case "grade":
		return runCramGrade(args[1:])
	default:
		printUsage()
		return 1
	}
}

func runCramAudioBackfill(args []string) int {
	fs := flag.NewFlagSet("cram audio-backfill", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	source := fs.String("source", "all", "all|hellochinese|travel_survival")
	voicesRaw := fs.String("voices", strings.Join(cram.DefaultVoices, ","), "comma-separated voices")
	ratesRaw := fs.String("rates", formatVoiceRates(cram.DefaultVoiceRates), "comma-separated voice=rate entries")
	concurrency := fs.Int("concurrency", 4, "parallel audio workers")
	limit := fs.Int("limit", 0, "max imported items to backfill; 0 means all")
	replace := fs.Bool("replace", false, "replace stored audio paths")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	sourceValue := strings.TrimSpace(*source)
	if sourceValue == "all" {
		sourceValue = ""
	}
	if sourceValue != "" && sourceValue != cram.SourceHelloChinese && sourceValue != cram.SourceTravelSurvival {
		return emitTypedError("cram.audio-backfill", "INVALID_ARGUMENT", "invalid source", map[string]any{"source": *source})
	}
	result, err := cram.BackfillAudio(cram.AudioBackfillOptions{
		Source:      sourceValue,
		Voices:      splitCSV(*voicesRaw),
		VoiceRates:  parseVoiceRates(*ratesRaw),
		Concurrency: *concurrency,
		Limit:       *limit,
		Replace:     *replace,
	})
	if err != nil {
		var toolMissing process.ToolMissingError
		if errors.As(err, &toolMissing) {
			return emitTypedError("cram.audio-backfill", "TOOL_MISSING", err.Error(), map[string]any{"tool": toolMissing.Tool})
		}
		var processFailed process.ProcessFailedError
		if errors.As(err, &processFailed) {
			return emitTypedError("cram.audio-backfill", "BACKEND_FAILED", "audio backend failed during cram backfill", map[string]any{
				"cmd":        processFailed.Cmd,
				"returncode": processFailed.ReturnCode,
				"stderr":     trim(processFailed.Stderr),
			})
		}
		return emitTypedError("cram.audio-backfill", "BACKEND_FAILED", err.Error(), nil)
	}
	out := envelope.OK("cram.audio-backfill", map[string]any{
		"tasks_seen":      result.TasksSeen,
		"audio_generated": result.AudioGenerated,
		"audio_existing":  result.AudioExisting,
		"audio_failed":    result.AudioFailed,
	}, nil, false, map[string]any{"source": *source, "concurrency": *concurrency, "limit": *limit, "replace": *replace})
	return emit(out)
}

func runCramOverview(args []string) int {
	fs := flag.NewFlagSet("cram overview", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	now, err := clock.NowUTC()
	if err != nil {
		return emitError("cram.overview", err)
	}
	overview, err := cram.OverviewFor(now)
	if err != nil {
		return emitError("cram.overview", err)
	}
	data := map[string]any{
		"generated_at": overview.GeneratedAt,
		"sources":      overview.Sources,
		"categories":   overview.Categories,
	}
	out := envelope.OK("cram.overview", data, nil, false, nil)
	return emit(out)
}

func runCramNext(args []string) int {
	fs := flag.NewFlagSet("cram next", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	limit := fs.Int("limit", 1, "limit")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	now, err := clock.NowUTC()
	if err != nil {
		return emitError("cram.next", err)
	}
	cards, err := cram.NextCards(cram.NextOptions{Limit: *limit}, now)
	if err != nil {
		return emitError("cram.next", err)
	}
	payload := []map[string]any{}
	for _, card := range cards {
		payload = append(payload, map[string]any{
			"item_id":              card.ItemID,
			"source":               card.Source,
			"category":             card.Category,
			"learning_order":       card.LearningOrder,
			"word":                 card.Word,
			"pinyin":               card.Pinyin,
			"meaning":              card.Meaning,
			"sentence_hanzi":       card.SentenceHanzi,
			"sentence_pinyin":      card.SentencePinyin,
			"sentence_meaning":     card.SentenceMeaning,
			"sentence_audio_paths": card.SentenceAudioPaths,
			"due_at":               card.DueAt,
			"unknown_other_words":  card.UnknownOtherWords,
		})
	}
	out := envelope.OK("cram.next", map[string]any{"cards": payload}, nil, false, map[string]any{"limit": *limit})
	return emit(out)
}

func runCramGrade(args []string) int {
	fs := flag.NewFlagSet("cram grade", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	item := fs.String("item", "", "item id")
	grade := fs.String("grade", "", "incorrect|correct")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	now, err := clock.NowUTC()
	if err != nil {
		return emitError("cram.grade", err)
	}
	result, err := cram.GradeCard(cram.GradeOptions{ItemID: *item, Grade: *grade}, now)
	if err != nil {
		return emitTypedError("cram.grade", "INVALID_ARGUMENT", err.Error(), map[string]any{"item": *item, "grade": *grade})
	}
	out := envelope.OK("cram.grade", map[string]any{
		"item_id":          result.ItemID,
		"grade":            result.Grade,
		"next_due_at":      result.NextDueAt,
		"interval_minutes": result.IntervalMinutes,
		"score":            result.Score,
		"difficulty":       result.Difficulty,
		"correct_count":    result.CorrectCount,
		"incorrect_count":  result.IncorrectCount,
		"reviewed_count":   result.ReviewedCount,
		"scored":           result.Scored,
	}, nil, false, nil)
	return emit(out)
}

func runVersion(args []string) int {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := fs.Bool("json", false, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *jsonOut {
		out := envelope.OK("version", map[string]any{"version": version}, nil, false, nil)
		return emit(out)
	}
	fmt.Fprintf(os.Stdout, "xuezh %s\n", version)
	return 0
}

func runSnapshot(args []string) int {
	fs := flag.NewFlagSet("snapshot", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	window := fs.String("window", "30d", "window")
	dueLimit := fs.Int("due-limit", 80, "due limit")
	evidenceLimit := fs.Int("evidence-limit", 200, "evidence limit")
	maxBytes := fs.Int("max-bytes", 200000, "max bytes")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	result, err := snapshot.BuildSnapshot(*window, *dueLimit, *evidenceLimit, *maxBytes)
	if err != nil {
		return emitError("snapshot", err)
	}
	out := envelope.OK("snapshot", result.Data, result.Artifacts, result.Truncated, result.Limits)
	return emit(out)
}

func runDB(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 1
	}
	switch args[0] {
	case "init":
		return runDBInit(args[1:])
	default:
		printUsage()
		return 1
	}
}

func runDBInit(args []string) int {
	fs := flag.NewFlagSet("db init", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	path, err := db.InitDB()
	if err != nil {
		return emitError("db.init", err)
	}
	out := envelope.OK("db.init", map[string]any{"db_path": path}, nil, false, nil)
	return emit(out)
}

func runDataset(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 1
	}
	switch args[0] {
	case "import":
		return runDatasetImport(args[1:])
	default:
		printUsage()
		return 1
	}
}

func runDatasetImport(args []string) int {
	fs := flag.NewFlagSet("dataset import", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	datasetType := fs.String("type", "", "hsk_vocab|hsk_chars|hsk_grammar|frequency")
	path := fs.String("path", "", "dataset path")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *datasetType == "" || *path == "" {
		return emitTypedError("dataset.import", "INVALID_ARGUMENT", "type and path are required", map[string]any{"type": *datasetType, "path": *path})
	}
	dsID, rows, err := datasets.ImportDataset(*datasetType, *path)
	if err != nil {
		return emitError("dataset.import", err)
	}
	out := envelope.OK("dataset.import", map[string]any{"type": *datasetType, "rows_loaded": rows, "dataset_id": dsID}, nil, false, nil)
	return emit(out)
}

func runReview(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 1
	}
	switch args[0] {
	case "start":
		return runReviewStart(args[1:])
	case "grade":
		return runReviewGrade(args[1:])
	case "bury":
		return runReviewBury(args[1:])
	default:
		printUsage()
		return 1
	}
}

func runReviewStart(args []string) int {
	fs := flag.NewFlagSet("review start", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	limit := fs.Int("limit", 10, "limit")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	now, err := clock.NowUTC()
	if err != nil {
		return emitError("review.start", err)
	}
	recallItems, err := srs.ListDueItems(*limit, now, "recall")
	if err != nil {
		return emitError("review.start", err)
	}
	pronunciationItems, err := srs.ListDueItems(*limit, now, "pronunciation")
	if err != nil {
		return emitError("review.start", err)
	}
	recallPayload := []map[string]any{}
	for _, item := range recallItems {
		recallPayload = append(recallPayload, map[string]any{"item_id": item.ItemID, "due_at": item.DueAt, "review_type": item.ReviewType})
	}
	pronPayload := []map[string]any{}
	for _, item := range pronunciationItems {
		pronPayload = append(pronPayload, map[string]any{"item_id": item.ItemID, "due_at": item.DueAt, "review_type": item.ReviewType})
	}
	out := envelope.OK("review.start", map[string]any{
		"items":               recallPayload,
		"recall_items":        recallPayload,
		"pronunciation_items": pronPayload,
		"generated_at":        clock.FormatISO(now),
	}, nil, false, map[string]any{"limit": *limit})
	return emit(out)
}

func runReviewGrade(args []string) int {
	fs := flag.NewFlagSet("review grade", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	item := fs.String("item", "", "item id")
	grade := fs.String("grade", "", "grade 0-5")
	recall := fs.String("recall", "", "recall 0-5")
	pronunciation := fs.String("pronunciation", "", "pronunciation 0-5")
	nextDue := fs.String("next-due", "", "next due")
	rule := fs.String("rule", "", "sm2|leitner")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *item == "" {
		return emitTypedError("review.grade", "INVALID_ARGUMENT", "item is required", map[string]any{"item": *item})
	}
	if *grade != "" && (*recall != "" || *pronunciation != "") {
		return emitTypedError("review.grade", "INVALID_ARGUMENT", "use --grade alone or --recall/--pronunciation, not both", map[string]any{"item": *item})
	}
	if *grade == "" && *recall == "" && *pronunciation == "" {
		return emitTypedError("review.grade", "INVALID_ARGUMENT", "provide --grade or --recall/--pronunciation", map[string]any{"item": *item})
	}
	parseInt := func(value string) (*int, error) {
		if value == "" {
			return nil, nil
		}
		v, err := strconv.Atoi(value)
		if err != nil {
			return nil, err
		}
		return &v, nil
	}
	recallGrade, err := parseInt(*recall)
	if err != nil {
		return emitTypedError("review.grade", "INVALID_ARGUMENT", "invalid recall grade", map[string]any{"item": *item})
	}
	pronGrade, err := parseInt(*pronunciation)
	if err != nil {
		return emitTypedError("review.grade", "INVALID_ARGUMENT", "invalid pronunciation grade", map[string]any{"item": *item})
	}
	if *grade != "" {
		value, err := strconv.Atoi(*grade)
		if err != nil {
			return emitTypedError("review.grade", "INVALID_ARGUMENT", "invalid grade", map[string]any{"item": *item})
		}
		recallGrade = &value
	}
	gradeValue := recallGrade
	if *grade == "" {
		gradeValue = nil
	}
	now, err := clock.NowUTC()
	if err != nil {
		return emitError("review.grade", err)
	}
	var recallDueAt *string
	var recallRule *string
	if recallGrade != nil {
		dueAt, appliedRule, err := srs.ScheduleNextDue(*recallGrade, now, *rule, *nextDue)
		if err != nil {
			return emitError("review.grade", err)
		}
		recallDueAt = &dueAt
		if appliedRule != "" {
			recallRule = &appliedRule
		}
	}
	var pronDueAt *string
	var pronRule *string
	if pronGrade != nil {
		pronNextDue := ""
		if recallGrade == nil {
			pronNextDue = *nextDue
		}
		dueAt, appliedRule, err := srs.ScheduleNextDue(*pronGrade, now, *rule, pronNextDue)
		if err != nil {
			return emitError("review.grade", err)
		}
		pronDueAt = &dueAt
		if appliedRule != "" {
			pronRule = &appliedRule
		}
	}
	if err := srs.UpsertKnowledge(*item, recallDueAt, recallGrade, pronDueAt, pronGrade, now); err != nil {
		return emitError("review.grade", err)
	}
	if recallGrade != nil {
		payload := map[string]any{
			"review_type": "recall",
			"grade":       *recallGrade,
			"rule":        recallRule,
			"next_due":    recallDueAt,
		}
		if err := srs.RecordReviewEvent(*item, "review.grade", payload, now); err != nil {
			return emitError("review.grade", err)
		}
	}
	if pronGrade != nil {
		payload := map[string]any{
			"review_type": "pronunciation",
			"grade":       *pronGrade,
			"rule":        pronRule,
			"next_due":    pronDueAt,
		}
		if err := srs.RecordReviewEvent(*item, "review.grade", payload, now); err != nil {
			return emitError("review.grade", err)
		}
	}
	data := map[string]any{"item": *item}
	if recallGrade != nil {
		data["recall_grade"] = *recallGrade
		data["recall_next_due"] = recallDueAt
		data["recall_rule_applied"] = recallRule
	}
	if pronGrade != nil {
		data["pronunciation_grade"] = *pronGrade
		data["pronunciation_next_due"] = pronDueAt
		data["pronunciation_rule_applied"] = pronRule
	}
	if gradeValue != nil {
		data["grade"] = *gradeValue
		data["next_due"] = recallDueAt
		data["rule_applied"] = recallRule
	}
	out := envelope.OK("review.grade", data, nil, false, nil)
	return emit(out)
}

func runReviewBury(args []string) int {
	fs := flag.NewFlagSet("review bury", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	item := fs.String("item", "", "item id")
	reason := fs.String("reason", "unspecified", "reason")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *item == "" {
		return emitTypedError("review.bury", "INVALID_ARGUMENT", "item is required", map[string]any{"item": *item})
	}
	now, err := clock.NowUTC()
	if err != nil {
		return emitError("review.bury", err)
	}
	dueAt, _, err := srs.ScheduleNextDue(0, now, "leitner", "")
	if err != nil {
		return emitError("review.bury", err)
	}
	if err := srs.UpsertKnowledge(*item, &dueAt, nil, nil, nil, now); err != nil {
		return emitError("review.bury", err)
	}
	payload := map[string]any{"reason": *reason, "next_due": dueAt}
	if err := srs.RecordReviewEvent(*item, "review.bury", payload, now); err != nil {
		return emitError("review.bury", err)
	}
	out := envelope.OK("review.bury", map[string]any{"item": *item, "reason": *reason, "next_due": dueAt}, nil, false, nil)
	return emit(out)
}

func runSRS(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 1
	}
	switch args[0] {
	case "preview":
		return runSRSPreview(args[1:])
	default:
		printUsage()
		return 1
	}
}

func runSRSPreview(args []string) int {
	fs := flag.NewFlagSet("srs preview", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	days := fs.Int("days", 14, "days")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	now, err := clock.NowUTC()
	if err != nil {
		return emitError("srs.preview", err)
	}
	recall, err := srs.PreviewDue(*days, now, "recall")
	if err != nil {
		return emitError("srs.preview", err)
	}
	pron, err := srs.PreviewDue(*days, now, "pronunciation")
	if err != nil {
		return emitError("srs.preview", err)
	}
	out := envelope.OK("srs.preview", map[string]any{"days": *days, "forecast": map[string]any{"recall": recall, "pronunciation": pron}}, nil, false, nil)
	return emit(out)
}

func runReport(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 1
	}
	switch args[0] {
	case "hsk":
		return runReportHSK(args[1:])
	case "mastery":
		return runReportMastery(args[1:])
	case "due":
		return runReportDue(args[1:])
	default:
		printUsage()
		return 1
	}
}

func runReportHSK(args []string) int {
	fs := flag.NewFlagSet("report hsk", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	level := fs.String("level", "", "level")
	window := fs.String("window", "30d", "window")
	maxItems := fs.Int("max-items", 200, "max items")
	maxBytes := fs.Int("max-bytes", 200000, "max bytes")
	includeChars := fs.Bool("include-chars", false, "include chars")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *level == "" {
		return emitTypedError("report.hsk", "INVALID_ARGUMENT", "level is required", map[string]any{"level": *level})
	}
	result, err := reports.BuildHSKReport(*level, *window, *maxItems, *maxBytes, *includeChars)
	if err != nil {
		return emitError("report.hsk", err)
	}
	out := envelope.OK("report.hsk", result.Data, result.Artifacts, result.Truncated, result.Limits)
	return emit(out)
}

func runReportMastery(args []string) int {
	fs := flag.NewFlagSet("report mastery", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	itemType := fs.String("item-type", "word", "word|character|grammar")
	window := fs.String("window", "90d", "window")
	maxItems := fs.Int("max-items", 200, "max items")
	maxBytes := fs.Int("max-bytes", 200000, "max bytes")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	result, err := reports.BuildMasteryReport(*itemType, *window, *maxItems, *maxBytes)
	if err != nil {
		return emitError("report.mastery", err)
	}
	out := envelope.OK("report.mastery", result.Data, result.Artifacts, result.Truncated, result.Limits)
	return emit(out)
}

func runReportDue(args []string) int {
	fs := flag.NewFlagSet("report due", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	limit := fs.Int("limit", 50, "limit")
	maxBytes := fs.Int("max-bytes", 200000, "max bytes")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	now, err := clock.NowUTC()
	if err != nil {
		return emitError("report.due", err)
	}
	items, err := srs.ListDueItems(*limit, now, "recall")
	if err != nil {
		return emitError("report.due", err)
	}
	payload := []map[string]any{}
	for _, item := range items {
		payload = append(payload, map[string]any{"item_id": item.ItemID, "due_at": item.DueAt})
	}
	out := envelope.OK("report.due", map[string]any{"items": payload}, nil, false, map[string]any{"limit": *limit, "max_bytes": *maxBytes})
	return emit(out)
}

func runEvent(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 1
	}
	switch args[0] {
	case "log":
		return runEventLog(args[1:])
	case "list":
		return runEventList(args[1:])
	default:
		printUsage()
		return 1
	}
}

func runEventLog(args []string) int {
	fs := flag.NewFlagSet("event log", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	eventType := fs.String("type", "", "exposure|review|pronunciation_attempt|content_served")
	modality := fs.String("modality", "", "reading|listening|speaking|typing|mixed")
	items := fs.String("items", "", "comma-separated item ids")
	itemsFile := fs.String("items-file", "", "file with item ids")
	context := fs.String("context", "", "context")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	parsed, err := events.ParseItems(*items, *itemsFile)
	if err != nil {
		return emitTypedError("event.log", "INVALID_ARGUMENT", err.Error(), map[string]any{"type": *eventType, "modality": *modality})
	}
	var contextPtr *string
	if *context != "" {
		contextPtr = context
	}
	event, err := events.LogEvent(*eventType, *modality, parsed, contextPtr)
	if err != nil {
		return emitError("event.log", err)
	}
	out := envelope.OK("event.log", map[string]any{
		"event_id":   event.EventID,
		"event_type": event.EventType,
		"ts":         event.TS,
		"modality":   event.Modality,
		"items":      event.Items,
		"context":    event.Context,
	}, nil, false, nil)
	return emit(out)
}

func runEventList(args []string) int {
	fs := flag.NewFlagSet("event list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	since := fs.String("since", "7d", "since")
	limit := fs.Int("limit", 200, "limit")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	items, err := events.ListEvents(*since, *limit)
	if err != nil {
		return emitError("event.list", err)
	}
	eventsPayload := []map[string]any{}
	for _, ev := range items {
		eventsPayload = append(eventsPayload, map[string]any{
			"event_id":   ev.EventID,
			"event_type": ev.EventType,
			"ts":         ev.TS,
			"modality":   ev.Modality,
			"items":      ev.Items,
			"context":    ev.Context,
		})
	}
	out := envelope.OK(
		"event.list",
		map[string]any{"events": eventsPayload},
		nil,
		false,
		map[string]any{"limit": *limit, "since": *since},
	)
	return emit(out)
}

func runContent(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 1
	}
	if args[0] != "cache" {
		printUsage()
		return 1
	}
	return runContentCache(args[1:])
}

func runContentCache(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 1
	}
	switch args[0] {
	case "put":
		return runContentCachePut(args[1:])
	case "get":
		return runContentCacheGet(args[1:])
	default:
		printUsage()
		return 1
	}
}

func runContentCachePut(args []string) int {
	fs := flag.NewFlagSet("content cache put", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	contentType := fs.String("type", "", "story|dialogue|exercise")
	key := fs.String("key", "", "key")
	inPath := fs.String("in", "", "input path")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	result, err := content.PutContent(*contentType, *key, *inPath)
	if err != nil {
		return emitTypedError("content.cache.put", "INVALID_ARGUMENT", err.Error(), map[string]any{"type": *contentType, "key": *key, "in": *inPath})
	}
	out := envelope.OK("content.cache.put", result.Data, result.Artifacts, false, nil)
	return emit(out)
}

func runContentCacheGet(args []string) int {
	fs := flag.NewFlagSet("content cache get", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	contentType := fs.String("type", "", "story|dialogue|exercise")
	key := fs.String("key", "", "key")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	result, err := content.GetContent(*contentType, *key)
	if err != nil {
		return emitTypedError("content.cache.get", "NOT_FOUND", err.Error(), map[string]any{"type": *contentType, "key": *key})
	}
	out := envelope.OK("content.cache.get", result.Data, result.Artifacts, false, nil)
	return emit(out)
}

func runAudio(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 1
	}
	switch args[0] {
	case "convert":
		return runAudioConvert(args[1:])
	case "tts":
		return runAudioTTS(args[1:])
	case "process-voice":
		return runAudioProcessVoice(args[1:])
	default:
		printUsage()
		return 1
	}
}

func runAudioConvert(args []string) int {
	fs := flag.NewFlagSet("audio convert", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	inPath := fs.String("in", "", "input path")
	outPath := fs.String("out", "", "output path")
	format := fs.String("format", "", "wav|ogg|mp3")
	backend := fs.String("backend", "", "backend")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *inPath == "" || *outPath == "" || *format == "" {
		return emitTypedError("audio.convert", "INVALID_ARGUMENT", "in, out, and format are required", map[string]any{"in": *inPath, "out": *outPath, "format": *format})
	}
	resolvedBackend := resolveAudioBackend(*backend, "ffmpeg", "XUEZH_AUDIO_CONVERT_BACKEND", "convert_backend")
	result, err := audio.ConvertAudio(*inPath, *outPath, *format, resolvedBackend, "converted_audio")
	if err != nil {
		var toolMissing process.ToolMissingError
		if errors.As(err, &toolMissing) {
			return emitTypedError(
				"audio.convert",
				"TOOL_MISSING",
				err.Error(),
				map[string]any{"tool": toolMissing.Tool, "in": *inPath, "out": *outPath, "format": *format, "backend": resolvedBackend},
			)
		}
		var processFailed process.ProcessFailedError
		if errors.As(err, &processFailed) {
			return emitTypedError(
				"audio.convert",
				"BACKEND_FAILED",
				"audio backend failed during conversion",
				map[string]any{
					"cmd":        processFailed.Cmd,
					"returncode": processFailed.ReturnCode,
					"stderr":     trim(processFailed.Stderr),
					"in":         *inPath,
					"out":        *outPath,
					"format":     *format,
					"backend":    resolvedBackend,
				},
			)
		}
		return emitTypedError(
			"audio.convert",
			"INVALID_ARGUMENT",
			err.Error(),
			map[string]any{"in": *inPath, "out": *outPath, "format": *format, "backend": resolvedBackend},
		)
	}
	out := envelope.OK("audio.convert", result.Data, result.Artifacts, false, nil)
	return emit(out)
}

func runAudioTTS(args []string) int {
	fs := flag.NewFlagSet("audio tts", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	text := fs.String("text", "", "text")
	voice := fs.String("voice", "XiaoxiaoNeural", "voice")
	outPath := fs.String("out", "", "output path")
	backend := fs.String("backend", "", "backend")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *text == "" || *outPath == "" {
		return emitTypedError("audio.tts", "INVALID_ARGUMENT", "text and out are required", map[string]any{"text": *text, "out": *outPath})
	}
	resolvedBackend := resolveAudioBackend(*backend, "edge-tts", "XUEZH_AUDIO_TTS_BACKEND", "tts_backend")
	result, err := audio.TTSAudio(*text, *voice, *outPath, resolvedBackend, "tts_audio")
	if err != nil {
		var toolMissing process.ToolMissingError
		if errors.As(err, &toolMissing) {
			return emitTypedError(
				"audio.tts",
				"TOOL_MISSING",
				err.Error(),
				map[string]any{"tool": toolMissing.Tool, "text": *text, "voice": *voice, "out": *outPath, "backend": resolvedBackend},
			)
		}
		var processFailed process.ProcessFailedError
		if errors.As(err, &processFailed) {
			return emitTypedError(
				"audio.tts",
				"BACKEND_FAILED",
				"audio backend failed during tts",
				map[string]any{
					"cmd":        processFailed.Cmd,
					"returncode": processFailed.ReturnCode,
					"stderr":     trim(processFailed.Stderr),
					"text":       *text,
					"voice":      *voice,
					"out":        *outPath,
					"backend":    resolvedBackend,
				},
			)
		}
		return emitTypedError(
			"audio.tts",
			"INVALID_ARGUMENT",
			err.Error(),
			map[string]any{"text": *text, "voice": *voice, "out": *outPath, "backend": resolvedBackend},
		)
	}
	out := envelope.OK("audio.tts", result.Data, result.Artifacts, false, nil)
	return emit(out)
}

func runAudioProcessVoice(args []string) int {
	fs := flag.NewFlagSet("audio process-voice", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	inPath := fs.String("in", "", "input path")
	refText := fs.String("ref-text", "", "reference text")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *inPath == "" || *refText == "" {
		return emitTypedError("audio.process-voice", "INVALID_ARGUMENT", "in and ref-text are required", map[string]any{"in": *inPath, "ref_text": *refText})
	}
	backend := resolveAudioBackend("", "azure.speech", "XUEZH_AUDIO_PROCESS_VOICE_BACKEND", "process_voice_backend")
	result, err := audio.ProcessVoice(*inPath, *refText, backend)
	if err != nil {
		var azureErr audio.AzureSpeechError
		if errors.As(err, &azureErr) {
			errorType := "BACKEND_FAILED"
			if azureErr.Kind == "quota" {
				errorType = "QUOTA_EXCEEDED"
			} else if azureErr.Kind == "auth" {
				errorType = "AUTH_FAILED"
			}
			details := map[string]any{"ref_text": *refText, "in": *inPath, "backend": backend}
			for key, value := range azureErr.Details {
				details[key] = value
			}
			return emitTypedError("audio.process-voice", errorType, azureErr.Error(), details)
		}
		var toolMissing process.ToolMissingError
		if errors.As(err, &toolMissing) {
			return emitTypedError(
				"audio.process-voice",
				"TOOL_MISSING",
				err.Error(),
				map[string]any{"tool": toolMissing.Tool, "ref_text": *refText, "in": *inPath, "backend": backend},
			)
		}
		var processFailed process.ProcessFailedError
		if errors.As(err, &processFailed) {
			return emitTypedError(
				"audio.process-voice",
				"BACKEND_FAILED",
				"audio backend failed during voice processing",
				map[string]any{
					"cmd":        processFailed.Cmd,
					"returncode": processFailed.ReturnCode,
					"stderr":     trim(processFailed.Stderr),
					"ref_text":   *refText,
					"in":         *inPath,
					"backend":    backend,
				},
			)
		}
		return emitTypedError(
			"audio.process-voice",
			"INVALID_ARGUMENT",
			err.Error(),
			map[string]any{"ref_text": *refText, "in": *inPath, "backend": backend},
		)
	}
	out := envelope.OK("audio.process-voice", result.Data, result.Artifacts, result.Truncated, result.Limits)
	return emit(out)
}

func runDoctor(args []string) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	checks := []map[string]any{}

	workspace, err := paths.WorkspaceDir()
	if err != nil {
		return emitError("doctor", err)
	}
	workspaceExists := false
	if _, err := os.Stat(workspace); err == nil {
		workspaceExists = true
	}
	workspaceOverride := os.Getenv("XUEZH_WORKSPACE_DIR")
	var workspaceOverrideValue any
	if workspaceOverride != "" {
		workspaceOverrideValue = workspaceOverride
	}
	checks = append(checks, map[string]any{
		"name": "workspace.path",
		"ok":   true,
		"details": map[string]any{
			"path":     workspace,
			"exists":   workspaceExists,
			"override": workspaceOverrideValue,
		},
	})

	dbPath, err := paths.DBPath()
	if err != nil {
		return emitError("doctor", err)
	}
	dbExists := false
	if _, err := os.Stat(dbPath); err == nil {
		dbExists = true
	}
	dbOverride := os.Getenv("XUEZH_DB_PATH")
	var dbOverrideValue any
	if dbOverride != "" {
		dbOverrideValue = dbOverride
	}
	dbDetails := map[string]any{"path": dbPath, "exists": dbExists, "override": dbOverrideValue}
	if dbExists {
		conn, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			dbDetails["error"] = err.Error()
			checks = append(checks, map[string]any{"name": "db.status", "ok": false, "details": dbDetails})
		} else {
			defer conn.Close()
			var count int
			row := conn.QueryRow("SELECT COUNT(*) FROM schema_migrations")
			if err := row.Scan(&count); err != nil {
				dbDetails["error"] = err.Error()
				checks = append(checks, map[string]any{"name": "db.status", "ok": false, "details": dbDetails})
			} else {
				dbDetails["schema_migrations"] = count
				checks = append(checks, map[string]any{"name": "db.status", "ok": true, "details": dbDetails})
			}
		}
	} else {
		checks = append(checks, map[string]any{"name": "db.status", "ok": false, "details": dbDetails})
	}

	for _, tool := range []string{"ffmpeg", "edge-tts", "whisper"} {
		path, err := exec.LookPath(tool)
		ok := err == nil && path != ""
		checks = append(checks, map[string]any{
			"name":    "tool." + tool,
			"ok":      ok,
			"details": map[string]any{"path": path},
		})
	}

	checks = append(checks, map[string]any{
		"name":    "tool.azure-speech-sdk",
		"ok":      true,
		"details": map[string]any{"version": "rest"},
	})

	configSection, ok, _ := config.GetValue("azure", "speech")
	var configKey, configRegion string
	var configKeyPresent bool
	if ok {
		if sectionMap, ok := configSection.(map[string]any); ok {
			if value, ok := sectionMap["key"].(string); ok && strings.TrimSpace(value) != "" {
				configKey = value
			}
			if value, ok := sectionMap["key_file"].(string); ok && strings.TrimSpace(value) != "" {
				keyPath := value
				if strings.HasPrefix(keyPath, "~") {
					if home, err := os.UserHomeDir(); err == nil {
						keyPath = filepath.Join(home, strings.TrimPrefix(keyPath, "~/"))
					}
				}
				if data, err := os.ReadFile(keyPath); err == nil {
					configKey = strings.TrimSpace(string(data))
				}
			}
			if value, ok := sectionMap["region"].(string); ok && strings.TrimSpace(value) != "" {
				configRegion = value
			}
		}
	}
	if strings.TrimSpace(configKey) != "" {
		configKeyPresent = true
	}
	envKeyPresent := os.Getenv("AZURE_SPEECH_KEY") != ""
	envRegionPresent := os.Getenv("AZURE_SPEECH_REGION") != ""
	configRegionPresent := strings.TrimSpace(configRegion) != ""
	configPath, _ := config.ConfigPath()
	checks = append(checks, map[string]any{
		"name": "azure.speech.env",
		"ok":   (envKeyPresent || configKeyPresent) && (envRegionPresent || configRegionPresent),
		"details": map[string]any{
			"AZURE_SPEECH_KEY":    envKeyPresent,
			"AZURE_SPEECH_REGION": envRegionPresent,
			"config_key":          configKeyPresent,
			"config_region":       configRegionPresent,
			"config_path":         configPath,
		},
	})

	out := envelope.OK("doctor", map[string]any{"checks": checks}, nil, false, nil)
	return emit(out)
}

func runGC(args []string) int {
	fs := flag.NewFlagSet("gc", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	apply := fs.Bool("apply", false, "apply deletions")
	dryRun := fs.Bool("dry-run", true, "preview deletions (default)")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *apply && *dryRun {
		*dryRun = false
	}
	if !*apply && !*dryRun {
		*dryRun = true
	}
	workspace, err := paths.EnsureWorkspace()
	if err != nil {
		return emitError("gc", err)
	}
	now, err := clock.NowUTC()
	if err != nil {
		return emitError("gc", err)
	}
	candidates, err := retention.CollectGCCandidates(workspace, now)
	if err != nil {
		return emitError("gc", err)
	}
	relCandidates := []string{}
	for _, path := range candidates {
		if rel, err := filepath.Rel(workspace, path); err == nil {
			relCandidates = append(relCandidates, rel)
		}
	}
	deletedCount := 0
	bytesFreed := int64(0)
	if *apply {
		for _, path := range candidates {
			info, err := os.Stat(path)
			if err != nil || info.IsDir() {
				continue
			}
			bytesFreed += info.Size()
			if err := os.Remove(path); err == nil {
				deletedCount++
			}
		}
	}
	out := envelope.OK(
		"gc",
		map[string]any{
			"dry_run":       *dryRun,
			"apply":         *apply,
			"candidates":    relCandidates,
			"deleted_count": deletedCount,
			"bytes_freed":   bytesFreed,
		},
		nil,
		false,
		nil,
	)
	return emit(out)
}

func resolveAudioBackend(cliValue, defaultValue, envKey, configKey string) string {
	if cliValue != "" {
		return cliValue
	}
	if configKey != "" {
		if value, ok := configString("audio", configKey); ok {
			return value
		}
	}
	if value, ok := configString("audio", "backend_global"); ok {
		return value
	}
	if envValue := os.Getenv(envKey); envValue != "" {
		return envValue
	}
	if envValue := os.Getenv("XUEZH_AUDIO_BACKEND"); envValue != "" {
		return envValue
	}
	return defaultValue
}

func configString(keys ...string) (string, bool) {
	value, ok, err := config.GetValue(keys...)
	if err != nil {
		return "", false
	}
	asString, ok := value.(string)
	if !ok {
		return "", false
	}
	asString = strings.TrimSpace(asString)
	if asString == "" {
		return "", false
	}
	return asString, true
}

func trim(text string) string {
	const limit = 2000
	if len(text) <= limit {
		return text
	}
	return text[:limit]
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := []string{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parseVoiceRates(value string) map[string]string {
	rates := map[string]string{}
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		pieces := strings.SplitN(part, "=", 2)
		if len(pieces) != 2 {
			continue
		}
		voice := strings.TrimSpace(pieces[0])
		rate := strings.TrimSpace(pieces[1])
		if voice != "" && rate != "" {
			rates[voice] = rate
		}
	}
	return rates
}

func formatVoiceRates(rates map[string]string) string {
	parts := []string{}
	for _, voice := range cram.DefaultVoices {
		if rate := rates[voice]; rate != "" {
			parts = append(parts, voice+"="+rate)
		}
	}
	return strings.Join(parts, ",")
}

func emit(payload any) int {
	body, err := jsonio.Dumps(payload)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Fprint(os.Stdout, body)
	return 0
}

func emitError(command string, err error) int {
	env, buildErr := envelope.Err(command, "BACKEND_FAILED", err.Error(), nil)
	if buildErr != nil {
		fmt.Fprintln(os.Stderr, buildErr)
		return 1
	}
	_ = emit(env)
	return 1
}

func emitTypedError(command, errorType, message string, details map[string]any) int {
	env, buildErr := envelope.Err(command, errorType, message, details)
	if buildErr != nil {
		fmt.Fprintln(os.Stderr, buildErr)
		return 1
	}
	_ = emit(env)
	return 1
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: xuezh <command> [args]")
	fmt.Fprintln(os.Stderr, "commands: version, snapshot, db, dataset, hellochinese, travel, pleco, cram, review, srs, report, event, content, audio, web, doctor, gc")
}

var ErrNotImplemented = errors.New("not implemented")
