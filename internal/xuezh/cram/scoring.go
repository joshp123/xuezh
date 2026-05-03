package cram

import (
	"database/sql"
	"math"
	"time"
)

const (
	defaultLearnedScore              = 200
	defaultMinScore                  = 100
	defaultMaxScore                  = 51200
	defaultMinDifficulty             = 50
	defaultMaxDifficulty             = 200
	defaultPointsPerDay              = 100
	defaultIncorrectScore            = 100
	defaultCorrectInitialScore       = 600
	defaultCorrectMultiplier         = 110
	defaultCorrectDifficultyChange   = 4
	defaultIncorrectDifficultyChange = -10
	defaultDifficultyDivisor         = 40
)

type scoringSettings struct {
	ProfileID                 int
	ProfileName               string
	LearnedScore              int
	MinScore                  int
	MaxScore                  int
	MinDifficulty             int
	MaxDifficulty             int
	PointsPerDay              int
	ScoreOncePerDay           bool
	IncorrectScore            int
	CorrectInitialScore       int
	CorrectMultiplier         int
	CorrectDifficultyChange   int
	IncorrectDifficultyChange int
	DifficultyDivisor         int
}

type scoreState struct {
	CardID          int
	Score           *int
	Difficulty      int
	History         string
	Correct         int
	Incorrect       int
	Reviewed        int
	FirstReviewedAt *time.Time
	LastReviewedAt  *time.Time
	SinceLastChange int
	ScoreIncTime    int64
	ScoreDecTime    int64
}

type scoreUpdate struct {
	Before scoreState
	After  scoreState
	Scored bool
}

type scoringSettingsQuerier interface {
	QueryRow(query string, args ...any) *sql.Row
}

func defaultScoringSettings() scoringSettings {
	return scoringSettings{
		ProfileID:                 2,
		ProfileName:               "Spaced Repetition",
		LearnedScore:              defaultLearnedScore,
		MinScore:                  defaultMinScore,
		MaxScore:                  defaultMaxScore,
		MinDifficulty:             defaultMinDifficulty,
		MaxDifficulty:             defaultMaxDifficulty,
		PointsPerDay:              defaultPointsPerDay,
		ScoreOncePerDay:           true,
		IncorrectScore:            defaultIncorrectScore,
		CorrectInitialScore:       defaultCorrectInitialScore,
		CorrectMultiplier:         defaultCorrectMultiplier,
		CorrectDifficultyChange:   defaultCorrectDifficultyChange,
		IncorrectDifficultyChange: defaultIncorrectDifficultyChange,
		DifficultyDivisor:         defaultDifficultyDivisor,
	}
}

func applyPlecoAnswer(before scoreState, settings scoringSettings, answer string, now time.Time) scoreUpdate {
	settings = normalizeScoringSettings(settings)
	after := before
	if after.Difficulty == 0 {
		after.Difficulty = 100
	}
	if after.FirstReviewedAt == nil {
		value := now
		after.FirstReviewedAt = &value
	}

	if settings.ScoreOncePerDay && before.LastReviewedAt != nil && sameLocalDay(*before.LastReviewedAt, now) {
		return scoreUpdate{Before: before, After: before, Scored: false}
	}

	value := now
	after.LastReviewedAt = &value
	after.Reviewed++
	after.SinceLastChange = 0
	if answer == GradeCorrect {
		after.Correct++
		after.History += "6"
		after.Difficulty = clampInt(after.Difficulty+settings.CorrectDifficultyChange, settings.MinDifficulty, settings.MaxDifficulty)
		oldScore := settings.MinScore
		if before.Score != nil && *before.Score > 0 {
			oldScore = *before.Score
		}
		next := int(math.Round(float64(oldScore) * float64(after.Difficulty) / float64(settings.DifficultyDivisor) * float64(settings.CorrectMultiplier) / 100.0))
		next = maxInt(next, settings.CorrectInitialScore)
		next = clampInt(next, settings.MinScore, settings.MaxScore)
		after.Score = &next
		after.ScoreIncTime = now.Unix()
		return scoreUpdate{Before: before, After: after, Scored: true}
	}

	after.Incorrect++
	after.History += "2"
	after.Difficulty = clampInt(after.Difficulty+settings.IncorrectDifficultyChange, settings.MinDifficulty, settings.MaxDifficulty)
	next := clampInt(settings.IncorrectScore, settings.MinScore, settings.MaxScore)
	after.Score = &next
	after.ScoreDecTime = now.Unix()
	return scoreUpdate{Before: before, After: after, Scored: true}
}

func normalizeScoringSettings(settings scoringSettings) scoringSettings {
	defaults := defaultScoringSettings()
	if settings.LearnedScore <= 0 {
		settings.LearnedScore = defaults.LearnedScore
	}
	if settings.MinScore <= 0 {
		settings.MinScore = defaults.MinScore
	}
	if settings.MaxScore <= 0 {
		settings.MaxScore = defaults.MaxScore
	}
	if settings.MinDifficulty <= 0 {
		settings.MinDifficulty = defaults.MinDifficulty
	}
	if settings.MaxDifficulty <= 0 {
		settings.MaxDifficulty = defaults.MaxDifficulty
	}
	if settings.PointsPerDay <= 0 {
		settings.PointsPerDay = defaults.PointsPerDay
	}
	if settings.IncorrectScore <= 0 {
		settings.IncorrectScore = defaults.IncorrectScore
	}
	if settings.CorrectInitialScore <= 0 {
		settings.CorrectInitialScore = defaults.CorrectInitialScore
	}
	if settings.CorrectMultiplier <= 0 {
		settings.CorrectMultiplier = defaults.CorrectMultiplier
	}
	if settings.CorrectDifficultyChange == 0 {
		settings.CorrectDifficultyChange = defaults.CorrectDifficultyChange
	}
	if settings.IncorrectDifficultyChange == 0 {
		settings.IncorrectDifficultyChange = defaults.IncorrectDifficultyChange
	}
	if settings.DifficultyDivisor <= 0 {
		settings.DifficultyDivisor = defaults.DifficultyDivisor
	}
	if settings.ProfileName == "" {
		settings.ProfileName = defaults.ProfileName
	}
	return settings
}

func intervalMinutesForScore(score int, settings scoringSettings) int {
	settings = normalizeScoringSettings(settings)
	minutes := int(math.Round(float64(score) / float64(settings.PointsPerDay) * 24 * 60))
	if minutes < 1 {
		return 1
	}
	return minutes
}

func loadLocalScoringSettings(conn scoringSettingsQuerier) (scoringSettings, error) {
	defaults := defaultScoringSettings()
	var settings scoringSettings
	var once int
	err := conn.QueryRow(`
		SELECT profile_id, profile_name, learned_score, min_score, max_score,
		       min_difficulty, max_difficulty, points_per_day, score_once_per_day,
		       incorrect_score, correct_initial_score, correct_multiplier,
		       correct_difficulty_change, incorrect_difficulty_change, difficulty_divisor
		FROM cram_pleco_scoring_settings
		WHERE id = 1`).
		Scan(
			&settings.ProfileID, &settings.ProfileName, &settings.LearnedScore,
			&settings.MinScore, &settings.MaxScore, &settings.MinDifficulty,
			&settings.MaxDifficulty, &settings.PointsPerDay, &once,
			&settings.IncorrectScore, &settings.CorrectInitialScore,
			&settings.CorrectMultiplier, &settings.CorrectDifficultyChange,
			&settings.IncorrectDifficultyChange, &settings.DifficultyDivisor,
		)
	if err == sql.ErrNoRows {
		return defaults, nil
	}
	if err != nil {
		return scoringSettings{}, err
	}
	settings.ScoreOncePerDay = once != 0
	return normalizeScoringSettings(settings), nil
}

func storeScoringSettings(tx *sql.Tx, settings scoringSettings, importedAt string) error {
	settings = normalizeScoringSettings(settings)
	once := 0
	if settings.ScoreOncePerDay {
		once = 1
	}
	_, err := tx.Exec(`
		INSERT INTO cram_pleco_scoring_settings (
		  id, profile_id, profile_name, learned_score, min_score, max_score,
		  min_difficulty, max_difficulty, points_per_day, score_once_per_day,
		  incorrect_score, correct_initial_score, correct_multiplier,
		  correct_difficulty_change, incorrect_difficulty_change, difficulty_divisor,
		  imported_at
		) VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
		  profile_id = excluded.profile_id,
		  profile_name = excluded.profile_name,
		  learned_score = excluded.learned_score,
		  min_score = excluded.min_score,
		  max_score = excluded.max_score,
		  min_difficulty = excluded.min_difficulty,
		  max_difficulty = excluded.max_difficulty,
		  points_per_day = excluded.points_per_day,
		  score_once_per_day = excluded.score_once_per_day,
		  incorrect_score = excluded.incorrect_score,
		  correct_initial_score = excluded.correct_initial_score,
		  correct_multiplier = excluded.correct_multiplier,
		  correct_difficulty_change = excluded.correct_difficulty_change,
		  incorrect_difficulty_change = excluded.incorrect_difficulty_change,
		  difficulty_divisor = excluded.difficulty_divisor,
		  imported_at = excluded.imported_at`,
		settings.ProfileID, settings.ProfileName, settings.LearnedScore,
		settings.MinScore, settings.MaxScore, settings.MinDifficulty,
		settings.MaxDifficulty, settings.PointsPerDay, once,
		settings.IncorrectScore, settings.CorrectInitialScore,
		settings.CorrectMultiplier, settings.CorrectDifficultyChange,
		settings.IncorrectDifficultyChange, settings.DifficultyDivisor,
		importedAt,
	)
	return err
}

func sameLocalDay(left, right time.Time) bool {
	ly, lm, ld := left.Local().Date()
	ry, rm, rd := right.Local().Date()
	return ly == ry && lm == rm && ld == rd
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
