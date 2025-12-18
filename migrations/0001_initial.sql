-- Initial schema (v0)
CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS characters (
  id TEXT PRIMARY KEY,
  character TEXT NOT NULL,
  pinyin TEXT,
  definition TEXT,
  source TEXT,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS words (
  id TEXT PRIMARY KEY,
  hanzi TEXT NOT NULL,
  pinyin TEXT,
  definition TEXT,
  source TEXT,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS grammar_points (
  id TEXT PRIMARY KEY,
  grammar_key TEXT,
  title TEXT,
  notes TEXT,
  source TEXT,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS user_knowledge (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  item_id TEXT NOT NULL,
  item_type TEXT NOT NULL,
  modality TEXT NOT NULL,
  first_seen_at TEXT,
  last_seen_at TEXT,
  seen_count INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS review_events (
  id TEXT PRIMARY KEY,
  item_id TEXT NOT NULL,
  event_type TEXT NOT NULL,
  ts TEXT NOT NULL,
  session_id TEXT,
  payload_json TEXT
);

CREATE TABLE IF NOT EXISTS learning_sessions (
  id TEXT PRIMARY KEY,
  started_at TEXT NOT NULL,
  ended_at TEXT,
  notes TEXT
);

CREATE TABLE IF NOT EXISTS pronunciation_attempts (
  id TEXT PRIMARY KEY,
  item_id TEXT,
  ts TEXT NOT NULL,
  backend_id TEXT,
  artifacts_json TEXT,
  summary_json TEXT
);

CREATE TABLE IF NOT EXISTS generated_content (
  id TEXT PRIMARY KEY,
  content_type TEXT NOT NULL,
  content_key TEXT NOT NULL,
  path TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS error_patterns (
  id TEXT PRIMARY KEY,
  pattern TEXT NOT NULL,
  notes TEXT,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS datasets (
  id TEXT PRIMARY KEY,
  dataset_type TEXT NOT NULL,
  version TEXT,
  source TEXT,
  ingested_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS dataset_items (
  dataset_id TEXT NOT NULL,
  item_id TEXT NOT NULL,
  item_type TEXT NOT NULL,
  payload_json TEXT,
  PRIMARY KEY (dataset_id, item_id)
);
