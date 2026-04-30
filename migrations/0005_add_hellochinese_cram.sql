CREATE TABLE IF NOT EXISTS hellochinese_items (
  id TEXT PRIMARY KEY,
  learning_order INTEGER NOT NULL UNIQUE,
  source_index INTEGER,
  unit_label TEXT,
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
  UNIQUE (hanzi, sentence_hanzi)
);

CREATE TABLE IF NOT EXISTS hellochinese_cram_state (
  item_id TEXT PRIMARY KEY,
  status TEXT NOT NULL,
  next_due_at TEXT,
  seen_count INTEGER NOT NULL DEFAULT 0,
  lapse_count INTEGER NOT NULL DEFAULT 0,
  last_grade TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (item_id) REFERENCES hellochinese_items(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_hellochinese_cram_due
ON hellochinese_cram_state(next_due_at, status);
