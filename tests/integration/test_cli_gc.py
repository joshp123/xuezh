import json
import os
import subprocess
import sys
from datetime import datetime, timedelta, timezone
from pathlib import Path


def _touch(path: Path, mtime: datetime) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text("x", encoding="utf-8")
    ts = mtime.timestamp()
    os.utime(path, (ts, ts))


def _run(env: dict[str, str], *args: str) -> dict:
    p = subprocess.run([sys.executable, "-m", "xuezh.cli", *args], env=env, capture_output=True, text=True)
    assert p.returncode == 0, p.stderr
    return json.loads(p.stdout)


def test_gc_dry_run_and_apply(tmp_path, monkeypatch):
    now = datetime(2025, 1, 10, tzinfo=timezone.utc)
    old_file = tmp_path / "artifacts" / "old.txt"
    new_file = tmp_path / "artifacts" / "new.txt"

    _touch(old_file, now - timedelta(days=5))
    _touch(new_file, now - timedelta(days=0))

    env = os.environ.copy()
    env["XUEZH_WORKSPACE_DIR"] = str(tmp_path)
    env["XUEZH_TEST_NOW_ISO"] = now.isoformat()
    env["XUEZH_RETENTION_ARTIFACTS_DAYS"] = "3"

    out = _run(env, "gc", "--dry-run", "--json")
    assert out["ok"] is True
    assert "artifacts/old.txt" in out["data"]["candidates"]
    assert old_file.exists()

    out = _run(env, "gc", "--apply", "--json")
    assert out["ok"] is True
    assert not old_file.exists()
    assert new_file.exists()
