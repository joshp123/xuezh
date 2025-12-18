import json
import os
import subprocess
import sys


def _run(env: dict[str, str], *args: str) -> dict:
    p = subprocess.run([sys.executable, "-m", "xuezh.cli", *args], env=env, capture_output=True, text=True)
    assert p.returncode == 0, p.stderr
    return json.loads(p.stdout)


def test_event_log_and_list(tmp_path):
    env = os.environ.copy()
    env["XUEZH_WORKSPACE_DIR"] = str(tmp_path)
    env["XUEZH_TEST_NOW_ISO"] = "2025-01-02T03:04:05+00:00"

    _run(env, "db", "init", "--json")
    log = _run(
        env,
        "event",
        "log",
        "--type",
        "exposure",
        "--modality",
        "reading",
        "--items",
        "w_aaaaaaaaaaaa",
        "--context",
        "ct_deadbeefcafe",
        "--json",
    )
    assert log["data"]["event_type"] == "exposure"

    listed = _run(env, "event", "list", "--since", "7d", "--limit", "200", "--json")
    assert listed["data"]["events"][0]["event_id"] == log["data"]["event_id"]
