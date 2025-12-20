import json
import os
import subprocess
import sys
from pathlib import Path


def _run(env: dict[str, str], *args: str) -> dict:
    p = subprocess.run([sys.executable, "-m", "xuezh.cli", *args], env=env, capture_output=True, text=True)
    assert p.returncode == 0, p.stderr
    return json.loads(p.stdout)


def test_report_hsk_with_fixtures(tmp_path):
    repo_root = Path(__file__).resolve().parents[2]
    vocab = repo_root / "tests" / "fixtures" / "datasets" / "hsk_vocab_min.csv"
    grammar = repo_root / "tests" / "fixtures" / "datasets" / "hsk_grammar_min.csv"

    env = os.environ.copy()
    env["XUEZH_WORKSPACE_DIR"] = str(tmp_path)
    env["XUEZH_TEST_NOW_ISO"] = "2025-01-02T03:04:05+00:00"

    _run(env, "db", "init", "--json")
    _run(env, "dataset", "import", "--type", "hsk_vocab", "--path", str(vocab), "--json")
    _run(env, "dataset", "import", "--type", "hsk_grammar", "--path", str(grammar), "--json")

    out = _run(
        env,
        "report",
        "hsk",
        "--level",
        "3",
        "--window",
        "30d",
        "--max-items",
        "200",
        "--max-bytes",
        "200000",
        "--json",
    )
    assert out["data"]["coverage"]["vocab"]["total"] == 5
    assert out["data"]["coverage"]["grammar"]["total"] == 2


def test_report_hsk_supports_7_9_bucket(tmp_path):
    repo_root = Path(__file__).resolve().parents[2]
    env = os.environ.copy()
    env["XUEZH_WORKSPACE_DIR"] = str(tmp_path)
    env["XUEZH_TEST_NOW_ISO"] = "2025-01-02T03:04:05+00:00"

    _run(env, "db", "init", "--json")

    vocab = repo_root / "tests" / "fixtures" / "datasets" / "hsk_vocab_7_9.csv"
    _run(env, "dataset", "import", "--type", "hsk_vocab", "--path", str(vocab), "--json")

    out = _run(
        env,
        "report",
        "hsk",
        "--level",
        "6",
        "--window",
        "30d",
        "--max-items",
        "200",
        "--max-bytes",
        "200000",
        "--json",
    )
    assert out["data"]["coverage"]["vocab"]["total"] == 1
    assert out["data"]["counts_by_level"]["vocab"]["7-9"]["total"] == 1

    out_full = _run(
        env,
        "report",
        "hsk",
        "--level",
        "9",
        "--window",
        "30d",
        "--max-items",
        "200",
        "--max-bytes",
        "200000",
        "--json",
    )
    assert out_full["data"]["coverage"]["vocab"]["total"] == 2
    assert out_full["data"]["counts_by_level"]["vocab"]["7-9"]["total"] == 1


def test_report_mastery_with_review(tmp_path):
    env = os.environ.copy()
    env["XUEZH_WORKSPACE_DIR"] = str(tmp_path)
    env["XUEZH_TEST_NOW_ISO"] = "2025-01-02T03:04:05+00:00"

    _run(env, "db", "init", "--json")
    _run(
        env,
        "review",
        "grade",
        "--item",
        "w_aaaaaaaaaaaa",
        "--grade",
        "4",
        "--next-due",
        "2025-01-03T03:04:05+00:00",
        "--json",
    )

    out = _run(
        env,
        "report",
        "mastery",
        "--item-type",
        "word",
        "--window",
        "90d",
        "--max-items",
        "50",
        "--max-bytes",
        "200000",
        "--json",
    )
    assert out["data"]["items"][0]["item_id"] == "w_aaaaaaaaaaaa"
