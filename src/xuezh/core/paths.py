from __future__ import annotations

import os
from pathlib import Path
from datetime import date

DEFAULT_WORKSPACE_DIR = Path("~/.clawdis/workspace/xuezh/").expanduser()
WORKSPACE_SUBDIRS = ("artifacts", "cache", "exports", "backups")


def workspace_dir() -> Path:
    override = os.environ.get("XUEZH_WORKSPACE_DIR")
    return Path(override).expanduser() if override else DEFAULT_WORKSPACE_DIR


def db_path() -> Path:
    override = os.environ.get("XUEZH_DB_PATH")
    if override:
        return Path(override).expanduser()
    return workspace_dir() / "db.sqlite3"


def ensure_workspace() -> Path:
    root = workspace_dir()
    root.mkdir(parents=True, exist_ok=True)
    for subdir in WORKSPACE_SUBDIRS:
        (root / subdir).mkdir(parents=True, exist_ok=True)
    return root


def resolve_in_workspace(path: str | Path) -> Path:
    root = ensure_workspace().resolve()
    candidate = Path(path).expanduser()
    resolved = candidate.resolve() if candidate.is_absolute() else (root / candidate).resolve()
    if not resolved.is_relative_to(root):
        raise ValueError(f"path escapes workspace: {path}")
    return resolved


def daily_backup_path(day: date) -> Path:
    root = ensure_workspace()
    filename = f"backup-{day.isoformat()}.sqlite3"
    return root / "backups" / filename
