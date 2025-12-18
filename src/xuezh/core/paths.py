from __future__ import annotations

import os
from pathlib import Path

DEFAULT_WORKSPACE_DIR = Path("~/.clawdis/workspace/xuezh/").expanduser()


def workspace_dir() -> Path:
    override = os.environ.get("XUEZH_WORKSPACE_DIR")
    return Path(override).expanduser() if override else DEFAULT_WORKSPACE_DIR


def db_path() -> Path:
    override = os.environ.get("XUEZH_DB_PATH")
    if override:
        return Path(override).expanduser()
    return workspace_dir() / "db.sqlite3"
