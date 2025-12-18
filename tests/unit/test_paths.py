from datetime import date

import pytest

from xuezh.core import paths


def test_ensure_workspace_creates_subdirs(tmp_path, monkeypatch):
    monkeypatch.setenv("XUEZH_WORKSPACE_DIR", str(tmp_path))
    root = paths.ensure_workspace()
    for subdir in paths.WORKSPACE_SUBDIRS:
        assert (root / subdir).is_dir()


def test_resolve_in_workspace_accepts_relative(tmp_path, monkeypatch):
    monkeypatch.setenv("XUEZH_WORKSPACE_DIR", str(tmp_path))
    resolved = paths.resolve_in_workspace("artifacts/output.wav")
    assert resolved == (tmp_path / "artifacts" / "output.wav").resolve()


def test_resolve_in_workspace_accepts_absolute_inside(tmp_path, monkeypatch):
    monkeypatch.setenv("XUEZH_WORKSPACE_DIR", str(tmp_path))
    target = tmp_path / "exports" / "file.json"
    resolved = paths.resolve_in_workspace(target)
    assert resolved == target.resolve()


def test_resolve_in_workspace_rejects_traversal(tmp_path, monkeypatch):
    monkeypatch.setenv("XUEZH_WORKSPACE_DIR", str(tmp_path))
    with pytest.raises(ValueError):
        paths.resolve_in_workspace("../escape.txt")


def test_daily_backup_path(tmp_path, monkeypatch):
    monkeypatch.setenv("XUEZH_WORKSPACE_DIR", str(tmp_path))
    expected = tmp_path / "backups" / "backup-2025-01-02.sqlite3"
    assert paths.daily_backup_path(date(2025, 1, 2)) == expected
