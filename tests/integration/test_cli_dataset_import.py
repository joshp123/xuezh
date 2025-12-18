import json
import os
import sqlite3
import subprocess
import sys
from pathlib import Path


def _run(env: dict[str, str], *args: str) -> dict:
    p = subprocess.run([sys.executable, "-m", "xuezh.cli", *args], env=env, capture_output=True, text=True)
    assert p.returncode == 0, p.stderr
    return json.loads(p.stdout)


def test_dataset_import_idempotent(tmp_path):
    repo_root = Path(__file__).resolve().parents[2]
    fixture = repo_root / "tests" / "fixtures" / "datasets" / "hsk_vocab_min.csv"

    env = os.environ.copy()
    env["XUEZH_WORKSPACE_DIR"] = str(tmp_path)

    out1 = _run(env, "dataset", "import", "--type", "hsk_vocab", "--path", str(fixture), "--json")
    out2 = _run(env, "dataset", "import", "--type", "hsk_vocab", "--path", str(fixture), "--json")

    assert out1["ok"] is True
    assert out1["data"]["rows_loaded"] == 5
    assert out2["data"]["dataset_id"] == out1["data"]["dataset_id"]

    db_path = tmp_path / "db.sqlite3"
    conn = sqlite3.connect(db_path)
    try:
        count = conn.execute(
            "SELECT COUNT(*) FROM dataset_items WHERE dataset_id = ?",
            (out1["data"]["dataset_id"],),
        ).fetchone()[0]
        assert count == 5
    finally:
        conn.close()
