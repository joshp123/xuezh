from __future__ import annotations

from dataclasses import dataclass
import mimetypes
from pathlib import Path
import shutil
import sqlite3

from xuezh.core import clock, db, ids, paths
from xuezh.core.envelope import Artifact

ALLOWED_CONTENT_TYPES = {"story", "dialogue", "exercise"}


@dataclass(frozen=True)
class ContentResult:
    data: dict
    artifacts: list[Artifact]


def _content_destination(content_type: str, key: str, suffix: str) -> Path:
    rel = Path("cache") / "content" / content_type / f"{key}{suffix}"
    return paths.resolve_in_workspace(rel)


def _artifact_for(path: Path) -> Artifact:
    workspace = paths.ensure_workspace()
    rel_path = str(path.relative_to(workspace))
    mime, _ = mimetypes.guess_type(path.name)
    return Artifact(path=rel_path, mime=mime or "text/plain", purpose="cached_content", bytes=path.stat().st_size)


def put_content(*, content_type: str, key: str, in_path: str) -> ContentResult:
    if content_type not in ALLOWED_CONTENT_TYPES:
        raise ValueError(f"Unsupported content type: {content_type}")

    input_path = Path(in_path).expanduser()
    if not input_path.exists():
        raise FileNotFoundError(f"Input file not found: {input_path}")

    suffix = input_path.suffix or ".txt"
    dest_path = _content_destination(content_type, key, suffix)
    dest_path.parent.mkdir(parents=True, exist_ok=True)

    content_id = ids.content_id(content_type=content_type, key=key)
    db_path = db.init_db()

    conn = sqlite3.connect(db_path)
    try:
        row = conn.execute(
            "SELECT path FROM generated_content WHERE id = ?",
            (content_id,),
        ).fetchone()

        if row:
            stored_rel = Path(row[0])
            existing_path = paths.resolve_in_workspace(stored_rel)
            if not existing_path.exists():
                shutil.copy2(input_path, existing_path)
            dest_path = existing_path
        else:
            if not dest_path.exists():
                shutil.copy2(input_path, dest_path)
            rel_path_str = str(dest_path.relative_to(paths.ensure_workspace()))
            conn.execute(
                """
                INSERT INTO generated_content (id, content_type, content_key, path, created_at)
                VALUES (?, ?, ?, ?, ?)
                """,
                (content_id, content_type, key, rel_path_str, clock.now_utc().isoformat()),
            )
            conn.commit()
    finally:
        conn.close()

    artifact = _artifact_for(dest_path)
    data = {"type": content_type, "key": key, "content_id": content_id}
    return ContentResult(data=data, artifacts=[artifact])


def get_content(*, content_type: str, key: str) -> ContentResult:
    if content_type not in ALLOWED_CONTENT_TYPES:
        raise ValueError(f"Unsupported content type: {content_type}")

    content_id = ids.content_id(content_type=content_type, key=key)
    db_path = db.init_db()

    conn = sqlite3.connect(db_path)
    try:
        row = conn.execute(
            "SELECT path FROM generated_content WHERE id = ?",
            (content_id,),
        ).fetchone()
        if not row:
            raise FileNotFoundError(f"Content not found for key: {key}")
        rel_path = Path(row[0])
        resolved = paths.resolve_in_workspace(rel_path)
        if not resolved.exists():
            raise FileNotFoundError(f"Cached content missing on disk: {resolved}")
    finally:
        conn.close()

    artifact = _artifact_for(resolved)
    data = {"type": content_type, "key": key, "content_id": content_id}
    return ContentResult(data=data, artifacts=[artifact])
