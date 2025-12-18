import json
import os
import sqlite3
import subprocess
import sys
from datetime import datetime, timezone

from xuezh.core import ids


def _run(env: dict[str, str], *args: str) -> dict:
    p = subprocess.run([sys.executable, "-m", "xuezh.cli", *args], env=env, capture_output=True, text=True)
    assert p.returncode == 0, p.stderr
    return json.loads(p.stdout)


def test_db_init_creates_schema(tmp_path):
    env = os.environ.copy()
    env["XUEZH_WORKSPACE_DIR"] = str(tmp_path)

    out = _run(env, "db", "init", "--json")
    assert out["ok"] is True
    db_path = tmp_path / "db.sqlite3"
    assert db_path.exists()

    conn = sqlite3.connect(db_path)
    try:
        tables = {row[0] for row in conn.execute("SELECT name FROM sqlite_master WHERE type='table'")}
        expected = {
            "schema_migrations",
            "users",
            "characters",
            "words",
            "grammar_points",
            "user_knowledge",
            "review_events",
            "learning_sessions",
            "pronunciation_attempts",
            "generated_content",
            "error_patterns",
            "datasets",
            "dataset_items",
        }
        assert expected.issubset(tables)

        user_id = "user_1"
        word = ids.word_id(hanzi="你好", pinyin="ni3 hao")
        event_id = ids.event_id_ulid()
        now = datetime(2025, 1, 2, tzinfo=timezone.utc).isoformat()

        conn.execute("INSERT INTO users (id, created_at) VALUES (?, ?)", (user_id, now))
        conn.execute(
            "INSERT INTO words (id, hanzi, pinyin, definition, source, created_at) VALUES (?, ?, ?, ?, ?, ?)",
            (word, "你好", "ni3 hao", "hello", "fixture", now),
        )
        conn.execute(
            "INSERT INTO review_events (id, item_id, event_type, ts, session_id, payload_json) VALUES (?, ?, ?, ?, ?, ?)",
            (event_id, word, "review.start", now, None, None),
        )
        conn.commit()
    finally:
        conn.close()
