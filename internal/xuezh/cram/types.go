package cram

type AudioGenerator func(text, voice, rate, outPath string) (string, error)

var DefaultVoices = []string{
	"zh-CN-XiaoxiaoNeural",
	"zh-CN-XiaoyiNeural",
	"zh-CN-YunxiNeural",
	"zh-CN-YunyangNeural",
}

var DefaultVoiceRates = map[string]string{
	"zh-CN-XiaoxiaoNeural": "-23%",
	"zh-CN-XiaoyiNeural":   "-15%",
	"zh-CN-YunxiNeural":    "-15%",
	"zh-CN-YunyangNeural":  "-25%",
}

const (
	SourceHelloChinese   = "hellochinese"
	SourceTravelSurvival = "travel_survival"
	GradeIncorrect       = "incorrect"
	GradeCorrect         = "correct"
)

type ImportOptions struct {
	Path           string
	AudioMode      string
	Voices         []string
	AudioGenerator AudioGenerator
}

type ImportResult struct {
	RowsSeen       int `json:"rows_seen"`
	RowsInserted   int `json:"rows_inserted"`
	RowsExisting   int `json:"rows_existing"`
	AudioGenerated int `json:"audio_generated"`
	AudioExisting  int `json:"audio_existing"`
	AudioFailed    int `json:"audio_failed"`
}

type PlecoScoreImportOptions struct {
	Path string
}

type PlecoScoreImportResult struct {
	CanonicalRows      int      `json:"canonical_rows"`
	SeededRows         int      `json:"seeded_rows"`
	UnseededRows       int      `json:"unseeded_rows"`
	UnseededCategories []string `json:"unseeded_categories"`
}

type Card struct {
	ItemID             string            `json:"item_id"`
	Source             string            `json:"source"`
	Category           string            `json:"category"`
	LearningOrder      int               `json:"learning_order"`
	Word               string            `json:"word"`
	Pinyin             string            `json:"pinyin"`
	Meaning            string            `json:"meaning"`
	SentenceHanzi      string            `json:"sentence_hanzi"`
	SentencePinyin     string            `json:"sentence_pinyin"`
	SentenceMeaning    string            `json:"sentence_meaning"`
	SentenceAudioPaths map[string]string `json:"sentence_audio_paths"`
	DueAt              *string           `json:"due_at"`
	UnknownOtherWords  *int              `json:"unknown_other_words"`
}

type CategoryRef struct {
	Source   string `json:"source"`
	Category string `json:"category"`
}

type NextOptions struct {
	Limit   int
	CardIDs []string
}

type GradeOptions struct {
	EventID    string
	ItemID     string
	Grade      string
	SessionID  string
	ShownAt    string
	AnsweredAt string
	ElapsedMS  int
	Round      int
	WasRetry   bool
}

type GradeResult struct {
	ItemID          string `json:"item_id"`
	Grade           string `json:"grade"`
	NextDueAt       string `json:"next_due_at"`
	IntervalMinutes int    `json:"interval_minutes"`
	Score           int    `json:"score"`
	Difficulty      int    `json:"difficulty"`
	CorrectCount    int    `json:"correct_count"`
	IncorrectCount  int    `json:"incorrect_count"`
	ReviewedCount   int    `json:"reviewed_count"`
	Scored          bool   `json:"scored"`
}

type ReviewSessionStartOptions struct {
	Limit   int
	CardIDs []string
}

type ReviewSessionState struct {
	ID            string                `json:"id"`
	Status        string                `json:"status"`
	Card          *Card                 `json:"card"`
	Queue         []Card                `json:"queue"`
	RetryQueue    []Card                `json:"retry_queue"`
	ReviewedCards []ReviewedSessionCard `json:"reviewed_cards"`
	RepeatItemIDs []string              `json:"repeat_item_ids"`
	Revealed      bool                  `json:"revealed"`
	Round         int                   `json:"round"`
	WasRetry      bool                  `json:"was_retry"`
	ShownAt       *string               `json:"shown_at"`
	UpdatedAt     string                `json:"updated_at"`
}

type ReviewedSessionCard struct {
	Card   Card   `json:"card"`
	Grade  string `json:"grade"`
	Repeat bool   `json:"repeat"`
}

type UndoGradeOptions struct {
	SessionID string
	ItemID    string
}

type UndoGradeResult struct {
	ItemID          string  `json:"item_id"`
	UndoneGrade     string  `json:"undone_grade"`
	NextDueAt       *string `json:"next_due_at"`
	IntervalMinutes int     `json:"interval_minutes"`
	Score           *int    `json:"score"`
	Difficulty      int     `json:"difficulty"`
	CorrectCount    int     `json:"correct_count"`
	IncorrectCount  int     `json:"incorrect_count"`
	ReviewedCount   int     `json:"reviewed_count"`
}

type AudioBackfillOptions struct {
	Source         string
	Voices         []string
	VoiceRates     map[string]string
	Concurrency    int
	Limit          int
	Replace        bool
	AudioGenerator AudioGenerator
}

type AudioBackfillResult struct {
	TasksSeen      int `json:"tasks_seen"`
	AudioGenerated int `json:"audio_generated"`
	AudioExisting  int `json:"audio_existing"`
	AudioFailed    int `json:"audio_failed"`
}

type Overview struct {
	GeneratedAt string            `json:"generated_at"`
	Sources     []SourceSummary   `json:"sources"`
	Categories  []CategorySummary `json:"categories"`
}

type PracticeFilters struct {
	ScoreBelow        int           `json:"score_below"`
	MissesMoreThan    int           `json:"misses_more_than"`
	IncludeNotLearned bool          `json:"include_not_learned"`
	IncludeDue        bool          `json:"include_due"`
	IncludeGotWrong   bool          `json:"include_got_wrong"`
	IncludeNoScore    bool          `json:"include_no_score"`
	Categories        []CategoryRef `json:"categories,omitempty"`
}

type PracticePreview struct {
	GeneratedAt string                    `json:"generated_at"`
	Sources     []PracticeSourceSummary   `json:"sources"`
	Categories  []PracticeCategorySummary `json:"categories"`
	Cards       []PracticeCard            `json:"cards"`
}

type PracticeSourceSummary struct {
	Source          string       `json:"source"`
	Label           string       `json:"label"`
	TotalCount      int          `json:"total_count"`
	PracticeCount   int          `json:"practice_count"`
	NotLearnedCount int          `json:"not_learned_count"`
	DueCount        int          `json:"due_count"`
	GotWrongCount   int          `json:"got_wrong_count"`
	ScoreBuckets    ScoreBuckets `json:"score_buckets"`
}

type PracticeCategorySummary struct {
	Source          string       `json:"source"`
	SourceLabel     string       `json:"source_label"`
	Category        string       `json:"category"`
	TotalCount      int          `json:"total_count"`
	PracticeCount   int          `json:"practice_count"`
	NotLearnedCount int          `json:"not_learned_count"`
	DueCount        int          `json:"due_count"`
	GotWrongCount   int          `json:"got_wrong_count"`
	ScoreBuckets    ScoreBuckets `json:"score_buckets"`
}

type ScoreBuckets struct {
	NoScore         int `json:"no_score"`
	ScoreUnder100   int `json:"score_under_100"`
	Score100To199   int `json:"score_100_to_199"`
	Score200To599   int `json:"score_200_to_599"`
	Score600To1599  int `json:"score_600_to_1599"`
	Score1600To6399 int `json:"score_1600_to_6399"`
	Score6400Plus   int `json:"score_6400_plus"`
}

type PracticeCard struct {
	ItemID         string  `json:"item_id"`
	Source         string  `json:"source"`
	SourceLabel    string  `json:"source_label"`
	Category       string  `json:"category"`
	LearningOrder  int     `json:"learning_order"`
	Word           string  `json:"word"`
	SentenceHanzi  string  `json:"sentence_hanzi"`
	Score          *int    `json:"score"`
	CorrectCount   int     `json:"correct_count"`
	IncorrectCount int     `json:"incorrect_count"`
	ReviewedCount  int     `json:"reviewed_count"`
	LastReviewedAt *string `json:"last_reviewed_at"`
	NotLearned     bool    `json:"not_learned"`
	Due            bool    `json:"due"`
	GotWrong       bool    `json:"got_wrong"`
}

type SourceSummary struct {
	Source       string `json:"source"`
	Label        string `json:"label"`
	TotalCount   int    `json:"total_count"`
	LearnedCount int    `json:"learned_count"`
	Eligible     int    `json:"eligible_count"`
}

type CategorySummary struct {
	Source       string `json:"source"`
	SourceLabel  string `json:"source_label"`
	Category     string `json:"category"`
	TotalCount   int    `json:"total_count"`
	LearnedCount int    `json:"learned_count"`
	Eligible     int    `json:"eligible_count"`
}

type OfflineDeckSnapshot struct {
	GeneratedAt string                 `json:"generated_at"`
	Cards       []OfflineDeckCard      `json:"cards"`
	AudioPaths  []string               `json:"audio_paths"`
	Settings    OfflineScoringSettings `json:"settings"`
}

type OfflineDeckCard struct {
	Card
	Score           *int    `json:"score"`
	Difficulty      int     `json:"difficulty"`
	CorrectCount    int     `json:"correct_count"`
	IncorrectCount  int     `json:"incorrect_count"`
	ReviewedCount   int     `json:"reviewed_count"`
	FirstReviewedAt *string `json:"first_reviewed_at"`
	LastReviewedAt  *string `json:"last_reviewed_at"`
}

type OfflineScoringSettings struct {
	LearnedScore              int  `json:"learned_score"`
	MinScore                  int  `json:"min_score"`
	MaxScore                  int  `json:"max_score"`
	MinDifficulty             int  `json:"min_difficulty"`
	MaxDifficulty             int  `json:"max_difficulty"`
	PointsPerDay              int  `json:"points_per_day"`
	ScoreOncePerDay           bool `json:"score_once_per_day"`
	IncorrectScore            int  `json:"incorrect_score"`
	CorrectInitialScore       int  `json:"correct_initial_score"`
	CorrectMultiplier         int  `json:"correct_multiplier"`
	CorrectDifficultyChange   int  `json:"correct_difficulty_change"`
	IncorrectDifficultyChange int  `json:"incorrect_difficulty_change"`
	DifficultyDivisor         int  `json:"difficulty_divisor"`
}

type OfflineReviewEvent struct {
	EventID    string `json:"event_id"`
	SessionID  string `json:"session_id"`
	ItemID     string `json:"item_id"`
	Grade      string `json:"grade"`
	ShownAt    string `json:"shown_at"`
	AnsweredAt string `json:"answered_at"`
	ElapsedMS  int    `json:"elapsed_ms"`
	Round      int    `json:"round"`
	WasRetry   bool   `json:"was_retry"`
}

type OfflineSyncResult struct {
	Applied         int      `json:"applied"`
	Skipped         int      `json:"skipped"`
	AppliedEventIDs []string `json:"applied_event_ids"`
	SkippedEventIDs []string `json:"skipped_event_ids"`
}

type itemRow struct {
	ID                 string
	Source             string
	Category           string
	LearningOrder      int
	SourceIndex        int
	Pinyin             string
	Hanzi              string
	Meaning            string
	SentencePinyin     string
	SentenceHanzi      string
	SentenceHanziRaw   string
	SentenceMeaning    string
	RowHash            string
	SentenceAudioPaths map[string]string
}
