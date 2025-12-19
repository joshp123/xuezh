import json
import os
import sqlite3
import subprocess
import sys
from pathlib import Path

from xuezh.core import ids, jsonio

GOLDEN_DIR = Path(__file__).resolve().parents[1] / "fixtures" / "golden"
FIXTURES = Path(__file__).resolve().parents[1] / "fixtures"


def _run(env: dict[str, str], *args: str) -> dict:
    p = subprocess.run([sys.executable, "-m", "xuezh.cli", *args], env=env, capture_output=True, text=True)
    assert p.returncode == 0, p.stderr
    return json.loads(p.stdout)


def _write_golden(name: str, payload: dict) -> None:
    path = GOLDEN_DIR / name
    path.write_text(jsonio.dumps(payload), encoding="utf-8")


def _load_golden(name: str) -> dict:
    path = GOLDEN_DIR / name
    return json.loads(path.read_text(encoding="utf-8"))


def _assert_or_update(name: str, payload: dict, update: bool) -> None:
    if update:
        _write_golden(name, payload)
        return
    expected = _load_golden(name)
    payload = _normalize_payload(name, payload)
    expected = _normalize_payload(name, expected)
    assert payload == expected


def _normalize_payload(name: str, payload: dict) -> dict:
    if name != "audio.process-voice.json":
        return payload
    # Audio tool output sizes can vary slightly between tool versions; ignore bytes.
    normalized = json.loads(json.dumps(payload))
    for artifact in normalized.get("artifacts", []):
        if str(artifact.get("mime", "")).startswith("audio/"):
            artifact.pop("bytes", None)
    return normalized


def _seed_db(db_path: Path) -> None:
    seed_path = FIXTURES / "db" / "seed_min.sql"
    sql = seed_path.read_text(encoding="utf-8")
    conn = sqlite3.connect(db_path)
    try:
        conn.executescript(sql)
        conn.commit()
    finally:
        conn.close()


def test_golden_traces(tmp_path):
    env = os.environ.copy()
    env["XUEZH_WORKSPACE_DIR"] = str(tmp_path)
    env["XUEZH_TEST_NOW_ISO"] = "2025-01-02T03:04:05+00:00"
    env["XUEZH_AUDIO_PROCESS_VOICE_BACKEND"] = "azure.speech"
    os.environ["XUEZH_WORKSPACE_DIR"] = str(tmp_path)
    os.environ["XUEZH_TEST_NOW_ISO"] = "2025-01-02T03:04:05+00:00"

    update = env.get("XUEZH_UPDATE_GOLDENS") == "1"

    _run(env, "db", "init", "--json")
    _run(env, "dataset", "import", "--type", "hsk_vocab", "--path", str(FIXTURES / "datasets" / "hsk_vocab_min.csv"), "--json")
    _run(env, "dataset", "import", "--type", "hsk_grammar", "--path", str(FIXTURES / "datasets" / "hsk_grammar_min.csv"), "--json")
    _run(env, "dataset", "import", "--type", "hsk_chars", "--path", str(FIXTURES / "datasets" / "hsk_chars_min.csv"), "--json")

    _seed_db(Path(env["XUEZH_WORKSPACE_DIR"]) / "db.sqlite3")

    item_id = ids.word_id(hanzi="你好", pinyin="nǐ hǎo")
    review_grade = _run(
        env,
        "review",
        "grade",
        "--item",
        item_id,
        "--grade",
        "4",
        "--next-due",
        env["XUEZH_TEST_NOW_ISO"],
        "--json",
    )
    review_start = _run(env, "review", "start", "--limit", "10", "--json")

    report_hsk = _run(env, "report", "hsk", "--level", "3", "--window", "30d", "--max-items", "50", "--max-bytes", "200000", "--json")
    snapshot = _run(env, "snapshot", "--window", "30d", "--due-limit", "80", "--evidence-limit", "200", "--max-bytes", "200000", "--json")
    process_voice = _run(
        env,
        "audio",
        "process-voice",
        "--in",
        str(FIXTURES / "audio" / "voice_sample.m4a"),
        "--ref-text",
        "你好",
        "--json",
    )

    _assert_or_update("review.grade.json", review_grade, update)
    _assert_or_update("review.start.json", review_start, update)
    _assert_or_update("report.hsk.json", report_hsk, update)
    _assert_or_update("snapshot.json", snapshot, update)
    _assert_or_update("audio.process-voice.json", process_voice, update)
