CREATE TABLE IF NOT EXISTS events (
  id TEXT PRIMARY KEY,
  event_type TEXT NOT NULL,
  ts TEXT NOT NULL,
  modality TEXT NOT NULL,
  items_json TEXT NOT NULL,
  context TEXT,
  payload_json TEXT
);
