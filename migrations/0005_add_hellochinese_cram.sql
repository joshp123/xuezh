CREATE TABLE IF NOT EXISTS cram_items (
  id TEXT PRIMARY KEY,
  source TEXT NOT NULL,
  category TEXT NOT NULL,
  learning_order INTEGER NOT NULL,
  source_index INTEGER,
  pinyin TEXT NOT NULL,
  hanzi TEXT NOT NULL,
  meaning TEXT NOT NULL,
  sentence_pinyin TEXT,
  sentence_hanzi TEXT NOT NULL,
  sentence_hanzi_raw TEXT,
  sentence_meaning TEXT,
  row_hash TEXT NOT NULL UNIQUE,
  sentence_audio_paths_json TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE (source, learning_order),
  UNIQUE (source, hanzi, sentence_hanzi)
);

CREATE TABLE IF NOT EXISTS cram_state (
  item_id TEXT PRIMARY KEY,
  next_due_at TEXT,
  pleco_card_id INTEGER,
  pleco_score INTEGER,
  pleco_difficulty INTEGER,
  pleco_history TEXT,
  pleco_correct_count INTEGER,
  pleco_incorrect_count INTEGER,
  pleco_reviewed_count INTEGER,
  pleco_first_reviewed_at TEXT,
  pleco_last_reviewed_at TEXT,
  pleco_sincelastchange INTEGER,
  pleco_score_inc_time INTEGER,
  pleco_score_dec_time INTEGER,
  score_imported_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (item_id) REFERENCES cram_items(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_cram_items_source_category
ON cram_items(source, category, learning_order);

CREATE INDEX IF NOT EXISTS idx_cram_state_due
ON cram_state(next_due_at);

CREATE TABLE IF NOT EXISTS cram_review_sessions (
  id TEXT PRIMARY KEY,
  status TEXT NOT NULL,
  current_item_id TEXT,
  queue_json TEXT NOT NULL DEFAULT '[]',
  retry_queue_json TEXT NOT NULL DEFAULT '[]',
  reviewed_json TEXT NOT NULL DEFAULT '[]',
  repeat_item_ids_json TEXT NOT NULL DEFAULT '[]',
  revealed INTEGER NOT NULL DEFAULT 0,
  round INTEGER NOT NULL DEFAULT 1,
  was_retry INTEGER NOT NULL DEFAULT 0,
  shown_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (current_item_id) REFERENCES cram_items(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_cram_review_sessions_active
ON cram_review_sessions(status, updated_at);

CREATE TABLE IF NOT EXISTS cram_pleco_scoring_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  profile_id INTEGER NOT NULL,
  profile_name TEXT NOT NULL,
  learned_score INTEGER NOT NULL,
  min_score INTEGER NOT NULL,
  max_score INTEGER NOT NULL,
  min_difficulty INTEGER NOT NULL,
  max_difficulty INTEGER NOT NULL,
  points_per_day INTEGER NOT NULL,
  score_once_per_day INTEGER NOT NULL,
  incorrect_score INTEGER NOT NULL,
  correct_initial_score INTEGER NOT NULL,
  correct_multiplier INTEGER NOT NULL,
  correct_difficulty_change INTEGER NOT NULL,
  incorrect_difficulty_change INTEGER NOT NULL,
  difficulty_divisor INTEGER NOT NULL,
  imported_at TEXT NOT NULL
);
