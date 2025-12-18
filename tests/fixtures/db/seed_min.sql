-- Minimal seed data for deterministic test runs
-- Assumes DB initialized via `xuezh db init`

INSERT OR IGNORE INTO words (id, hanzi, pinyin, definition, source, created_at)
VALUES ('w_0fafdf1e6a67', '你好', 'nǐ hǎo', 'hello|hi', 'seed', '2025-01-02T03:04:05+00:00');

INSERT OR IGNORE INTO user_knowledge
(item_id, item_type, modality, first_seen_at, last_seen_at, seen_count, due_at, last_grade)
VALUES (
  'w_0fafdf1e6a67',
  'word',
  'unknown',
  '2025-01-02T03:04:05+00:00',
  '2025-01-02T03:04:05+00:00',
  1,
  '2025-01-02T03:04:05+00:00',
  4
);
