import json
import os
import subprocess
import sys


def _run(env: dict[str, str], *args: str) -> dict:
    p = subprocess.run([sys.executable, "-m", "xuezh.cli", *args], env=env, capture_output=True, text=True)
    assert p.returncode == 0, p.stderr
    return json.loads(p.stdout)


def test_review_flow_due_and_preview(tmp_path):
    env = os.environ.copy()
    env["XUEZH_WORKSPACE_DIR"] = str(tmp_path)
    env["XUEZH_TEST_NOW_ISO"] = "2025-01-02T03:04:05+00:00"

    _run(env, "db", "init", "--json")

    # Grade an item with explicit next_due = now so it is due immediately.
    _run(
        env,
        "review",
        "grade",
        "--item",
        "w_aaaaaaaaaaaa",
        "--grade",
        "4",
        "--next-due",
        "2025-01-02T03:04:05+00:00",
        "--json",
    )

    start = _run(env, "review", "start", "--limit", "10", "--json")
    assert start["data"]["recall_items"][0]["item_id"] == "w_aaaaaaaaaaaa"
    assert "pronunciation_items" in start["data"]

    due = _run(env, "report", "due", "--limit", "10", "--max-bytes", "200000", "--json")
    assert due["data"]["items"][0]["item_id"] == "w_aaaaaaaaaaaa"

    preview = _run(env, "srs", "preview", "--days", "7", "--json")
    assert preview["data"]["days"] == 7
    assert "recall" in preview["data"]["forecast"]
    assert "pronunciation" in preview["data"]["forecast"]
