import json
import os
import subprocess
import sys
from pathlib import Path


def _run(env: dict[str, str], *args: str) -> dict:
    p = subprocess.run([sys.executable, "-m", "xuezh.cli", *args], env=env, capture_output=True, text=True)
    assert p.returncode == 0, p.stderr
    return json.loads(p.stdout)


def test_snapshot_deterministic_with_fixture(tmp_path):
    repo_root = Path(__file__).resolve().parents[2]
    vocab = repo_root / "tests" / "fixtures" / "datasets" / "hsk_vocab_min.csv"
    grammar = repo_root / "tests" / "fixtures" / "datasets" / "hsk_grammar_min.csv"

    env = os.environ.copy()
    env["XUEZH_WORKSPACE_DIR"] = str(tmp_path)
    env["XUEZH_TEST_NOW_ISO"] = "2025-01-02T03:04:05+00:00"

    _run(env, "db", "init", "--json")
    _run(env, "dataset", "import", "--type", "hsk_vocab", "--path", str(vocab), "--json")
    _run(env, "dataset", "import", "--type", "hsk_grammar", "--path", str(grammar), "--json")

    first = _run(
        env,
        "snapshot",
        "--window",
        "30d",
        "--due-limit",
        "20",
        "--evidence-limit",
        "50",
        "--max-bytes",
        "200000",
        "--json",
    )
    second = _run(
        env,
        "snapshot",
        "--window",
        "30d",
        "--due-limit",
        "20",
        "--evidence-limit",
        "50",
        "--max-bytes",
        "200000",
        "--json",
    )

    assert first == second
    assert first["data"]["window"] == "30d"
    assert first["data"]["generated_at"] == "2025-01-02T03:04:05+00:00"
    assert first["data"]["hsk_summary"]["vocab"]["total"] == 5
    assert first["data"]["hsk_summary"]["grammar"]["total"] == 2
