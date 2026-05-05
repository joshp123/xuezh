export type Card = {
  item_id: string;
  source: string;
  category: string;
  learning_order: number;
  word: string;
  pinyin: string;
  meaning: string;
  sentence_hanzi: string;
  sentence_pinyin: string;
  sentence_meaning: string;
  sentence_audio_paths: Record<string, string>;
};

export type SourceSummary = { source: string; label: string; total_count: number };
export type Overview = { generated_at: string; sources: SourceSummary[] };

export type ScoreBuckets = {
  no_score: number;
  score_under_100: number;
  score_100_to_199: number;
  score_200_to_599: number;
  score_600_to_1599: number;
  score_1600_to_6399: number;
  score_6400_plus: number;
};

export type PracticeSource = {
  source: string;
  label: string;
  total_count: number;
  practice_count: number;
  not_learned_count: number;
  due_count: number;
  got_wrong_count: number;
  score_buckets: ScoreBuckets;
};

export type PracticeCategory = {
  source: string;
  source_label: string;
  category: string;
  total_count: number;
  practice_count: number;
  not_learned_count: number;
  due_count: number;
  got_wrong_count: number;
  score_buckets: ScoreBuckets;
};

export type PracticeCard = {
  item_id: string;
  source: string;
  source_label: string;
  category: string;
  learning_order: number;
  word: string;
  sentence_hanzi: string;
  score: number | null;
  correct_count: number;
  incorrect_count: number;
  reviewed_count: number;
  last_reviewed_at: string | null;
  not_learned: boolean;
  due: boolean;
  got_wrong: boolean;
};

export type PracticePreview = {
  generated_at: string;
  sources: PracticeSource[];
  categories: PracticeCategory[];
  cards: PracticeCard[];
};

export type Filters = {
  score_below: number;
  misses_more_than: number;
  include_not_learned: boolean;
  include_due: boolean;
  include_got_wrong: boolean;
  include_no_score: boolean;
};

export type View = "batch" | "review" | "done";
export type ReviewAnswer = "incorrect" | "correct";
export type ReviewedCard = { card: Card; grade: ReviewAnswer; repeat: boolean };
export type ReviewSessionState = {
  id: string;
  status: "active" | "done";
  card: Card | null;
  queue: Card[];
  retry_queue: Card[];
  reviewed_cards: ReviewedCard[];
  repeat_item_ids: string[];
  revealed: boolean;
  round: number;
  was_retry: boolean;
  shown_at: string | null;
  updated_at: string;
};

export type OfflineDeckCard = Card & {
  score: number | null;
  difficulty: number;
  correct_count: number;
  incorrect_count: number;
  reviewed_count: number;
  first_reviewed_at: string | null;
  last_reviewed_at: string | null;
  due_at: string | null;
};

export type OfflineDeckSnapshot = {
  generated_at: string;
  cards: OfflineDeckCard[];
  audio_paths: string[];
  settings: {
    learned_score: number;
    min_score: number;
    max_score: number;
    min_difficulty: number;
    max_difficulty: number;
    points_per_day: number;
    score_once_per_day: boolean;
    incorrect_score: number;
    correct_initial_score: number;
    correct_multiplier: number;
    correct_difficulty_change: number;
    incorrect_difficulty_change: number;
    difficulty_divisor: number;
  };
};

export type OfflineReviewEvent = {
  event_id: string;
  session_id: string;
  item_id: string;
  grade: ReviewAnswer;
  shown_at: string;
  answered_at: string;
  elapsed_ms: number;
  round: number;
  was_retry: boolean;
};

export type OfflineSyncResult = {
  applied: number;
  skipped: number;
  applied_event_ids: string[];
  skipped_event_ids: string[];
};

export type OfflineStorageInfo = {
  persisted: boolean | null;
  usage_bytes: number | null;
  quota_bytes: number | null;
};

export type OfflineSaveProgress = {
  done: number;
  total: number;
  saved: number;
  missing: number;
};

export type OfflineSaveState = {
  saved_at: string;
  card_count: number;
  audio_total: number;
  audio_saved: number;
  audio_missing: number;
  storage: OfflineStorageInfo;
};

export const capOptions = [50, 100, 200, 500, 0];
export const defaultFilters: Filters = {
  score_below: 200,
  misses_more_than: 0,
  include_not_learned: true,
  include_due: true,
  include_got_wrong: false,
  include_no_score: true
};
