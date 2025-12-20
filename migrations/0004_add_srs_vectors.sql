ALTER TABLE user_knowledge ADD COLUMN recall_due_at TEXT;
ALTER TABLE user_knowledge ADD COLUMN recall_last_grade INTEGER;
ALTER TABLE user_knowledge ADD COLUMN pronunciation_due_at TEXT;
ALTER TABLE user_knowledge ADD COLUMN pronunciation_last_grade INTEGER;

UPDATE user_knowledge
SET recall_due_at = COALESCE(recall_due_at, due_at);

UPDATE user_knowledge
SET recall_last_grade = COALESCE(recall_last_grade, last_grade);
