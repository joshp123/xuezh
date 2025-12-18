from __future__ import annotations

import sqlite3
from pathlib import Path
from typing import Iterable

from xuezh.core import clock, paths


def _repo_root() -> Path:
    return Path(__file__).resolve().parents[3]


def _migrations_dir() -> Path:
    return _repo_root() / "migrations"


def _load_migrations() -> list[tuple[str, str]]:
    migrations_dir = _migrations_dir()
    if not migrations_dir.exists():
        raise FileNotFoundError(f"Missing migrations directory: {migrations_dir}")
    migrations: list[tuple[str, str]] = []
    for path in sorted(migrations_dir.glob("*.sql")):
        migrations.append((path.name, path.read_text(encoding="utf-8")))
    return migrations


def _ensure_migrations_table(conn: sqlite3.Connection) -> None:
    conn.execute(
        """
        CREATE TABLE IF NOT EXISTS schema_migrations (
          version TEXT PRIMARY KEY,
          applied_at TEXT NOT NULL
        )
        """
    )


def _applied_versions(conn: sqlite3.Connection) -> set[str]:
    rows = conn.execute("SELECT version FROM schema_migrations").fetchall()
    return {row[0] for row in rows}


def _apply_migration(conn: sqlite3.Connection, *, version: str, sql: str) -> None:
    conn.executescript(sql)
    conn.execute(
        "INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)",
        (version, clock.now_utc().isoformat()),
    )


def init_db() -> Path:
    db_path = paths.resolve_in_workspace(paths.db_path())
    db_path.parent.mkdir(parents=True, exist_ok=True)

    conn = sqlite3.connect(db_path)
    try:
        conn.execute("PRAGMA foreign_keys = ON;")
        _ensure_migrations_table(conn)
        applied = _applied_versions(conn)
        for version, sql in _load_migrations():
            if version in applied:
                continue
            _apply_migration(conn, version=version, sql=sql)
        conn.commit()
    finally:
        conn.close()

    return db_path


def list_tables(db_path: Path) -> Iterable[str]:
    conn = sqlite3.connect(db_path)
    try:
        rows = conn.execute("SELECT name FROM sqlite_master WHERE type='table'").fetchall()
        return [row[0] for row in rows]
    finally:
        conn.close()
