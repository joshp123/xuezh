from __future__ import annotations

import os
from dataclasses import dataclass
from datetime import datetime, timedelta, timezone
from pathlib import Path

from xuezh.core import paths

RETENTION_DEFAULTS = {
    "artifacts": 90,
    "backups": 30,
    "exports": 180,
    "cache": 180,
}

RETENTION_ENV = {
    "artifacts": "XUEZH_RETENTION_ARTIFACTS_DAYS",
    "backups": "XUEZH_RETENTION_BACKUPS_DAYS",
    "exports": "XUEZH_RETENTION_EXPORTS_DAYS",
    "cache": "XUEZH_RETENTION_CACHE_DAYS",
}


@dataclass(frozen=True)
class RetentionConfig:
    artifacts_days: int
    backups_days: int
    exports_days: int
    cache_days: int


def load_retention_config() -> RetentionConfig:
    def _days(key: str) -> int:
        env_key = RETENTION_ENV[key]
        raw = os.environ.get(env_key)
        if raw is None:
            return RETENTION_DEFAULTS[key]
        return int(raw)

    return RetentionConfig(
        artifacts_days=_days("artifacts"),
        backups_days=_days("backups"),
        exports_days=_days("exports"),
        cache_days=_days("cache"),
    )


def _files_under(root: Path, subdir: str) -> list[Path]:
    base = root / subdir
    if not base.exists():
        return []
    return sorted([p for p in base.rglob("*") if p.is_file()], key=lambda p: str(p))


def _is_older_than(path: Path, cutoff: datetime) -> bool:
    mtime = datetime.fromtimestamp(path.stat().st_mtime, tz=timezone.utc)
    return mtime < cutoff


def collect_gc_candidates(root: Path, *, now: datetime | None = None) -> list[Path]:
    now = now or datetime.now(timezone.utc)
    config = load_retention_config()

    candidates: list[Path] = []
    dir_windows = {
        "artifacts": config.artifacts_days,
        "backups": config.backups_days,
        "exports": config.exports_days,
        "cache": config.cache_days,
    }

    for subdir, days in dir_windows.items():
        cutoff = now - timedelta(days=days)
        for path in _files_under(root, subdir):
            if _is_older_than(path, cutoff):
                # Resolve relative to workspace and ensure safety.
                candidates.append(paths.resolve_in_workspace(path))

    # Deterministic ordering by workspace-relative path.
    return sorted(candidates, key=lambda p: str(p))
