from datetime import datetime, timedelta, timezone

import os

from xuezh.core import paths, retention


def _touch(path, mtime):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text("x", encoding="utf-8")
    ts = mtime.timestamp()
    os.utime(path, (ts, ts))


def test_collect_gc_candidates_respects_retention(tmp_path, monkeypatch):
    monkeypatch.setenv("XUEZH_WORKSPACE_DIR", str(tmp_path))
    now = datetime(2025, 1, 10, tzinfo=timezone.utc)

    old_artifact = tmp_path / "artifacts" / "2024" / "10" / "01" / "ar_old.txt"
    new_artifact = tmp_path / "artifacts" / "2025" / "01" / "09" / "ar_new.txt"
    old_export = tmp_path / "exports" / "export_old.json"

    _touch(old_artifact, now - timedelta(days=120))
    _touch(new_artifact, now - timedelta(days=2))
    _touch(old_export, now - timedelta(days=200))

    candidates = retention.collect_gc_candidates(paths.ensure_workspace(), now=now)
    rel = {str(p.relative_to(tmp_path)) for p in candidates}

    assert "artifacts/2024/10/01/ar_old.txt" in rel
    assert "exports/export_old.json" in rel
    assert "artifacts/2025/01/09/ar_new.txt" not in rel
