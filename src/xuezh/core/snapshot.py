from __future__ import annotations

import json
import sqlite3
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path

from xuezh.core import clock, db, events, jsonio, paths
from xuezh.core.envelope import Artifact


@dataclass(frozen=True)
class SnapshotResult:
    data: dict
    artifacts: list[Artifact]
    truncated: bool
    limits: dict


def _latest_dataset_id(conn: sqlite3.Connection, dataset_type: str) -> str | None:
    row = conn.execute(
        "SELECT id FROM datasets WHERE dataset_type = ? ORDER BY ingested_at DESC LIMIT 1",
        (dataset_type,),
    ).fetchone()
    return row[0] if row else None


def _counts_by_level(conn: sqlite3.Connection, dataset_id: str) -> dict[str, int]:
    counts: dict[str, int] = {}
    for (payload_json,) in conn.execute(
        "SELECT payload_json FROM dataset_items WHERE dataset_id = ?",
        (dataset_id,),
    ):
        payload = json.loads(payload_json)
        level = str(payload.get("hsk_level"))
        counts[level] = counts.get(level, 0) + 1
    return counts


def _summary(counts: dict[str, int]) -> dict[str, int]:
    total = sum(counts.values())
    return {"total": total}


def _artifact_path(prefix: str, now: datetime) -> Path:
    root = paths.ensure_workspace()
    day_path = root / "artifacts" / now.strftime("%Y") / now.strftime("%m") / now.strftime("%d")
    day_path.mkdir(parents=True, exist_ok=True)
    filename = f"{prefix}-{now.strftime('%Y%m%dT%H%M%SZ')}.json"
    return day_path / filename


def build_snapshot(
    *,
    window: str,
    due_limit: int,
    evidence_limit: int,
    max_bytes: int,
) -> SnapshotResult:
    now = clock.now_utc()
    workspace = paths.ensure_workspace()
    db_path = db.init_db()

    hsk_summary: dict[str, dict] = {}
    counts_by_level: dict[str, dict] = {}

    conn = sqlite3.connect(db_path)
    try:
        for dataset_type, key in (
            ("hsk_vocab", "vocab"),
            ("hsk_grammar", "grammar"),
            ("hsk_chars", "chars"),
        ):
            dataset_id = _latest_dataset_id(conn, dataset_type)
            if not dataset_id:
                continue
            counts = _counts_by_level(conn, dataset_id)
            hsk_summary[key] = _summary(counts)
            counts_by_level[key] = counts
    finally:
        conn.close()

    recent_events = events.list_events(since=window, limit=evidence_limit)
    exposure_counts = events.exposure_counts(since=window)

    data = {
        "generated_at": now.isoformat(),
        "window": window,
        "recent_events": [
            {
                "event_id": event.event_id,
                "event_type": event.event_type,
                "ts": event.ts,
                "modality": event.modality,
                "items": event.items,
                "context": event.context,
            }
            for event in recent_events
        ],
        "exposure_counts": exposure_counts,
        "due_items": [],
        "due_counts_by_day": {},
        "hsk_summary": hsk_summary,
        "counts_by_level": counts_by_level,
        "limits": {
            "due_limit": due_limit,
            "evidence_limit": evidence_limit,
        },
    }

    # Build full envelope to measure size.
    envelope = {
        "ok": True,
        "schema_version": "1",
        "command": "snapshot",
        "data": data,
        "artifacts": [],
        "truncated": False,
        "limits": {"max_bytes": max_bytes},
    }
    payload = jsonio.dumps(envelope)
    if len(payload.encode("utf-8")) <= max_bytes:
        return SnapshotResult(
            data=data,
            artifacts=[],
            truncated=False,
            limits={"max_bytes": max_bytes},
        )

    spill_path = _artifact_path("snapshot", now)
    spill_path.write_text(payload, encoding="utf-8")
    rel_path = str(spill_path.relative_to(workspace))
    artifact = Artifact(path=rel_path, mime="application/json", purpose="snapshot_spill", bytes=spill_path.stat().st_size)

    truncated_data = {
        "generated_at": now.isoformat(),
        "window": window,
        "recent_events": [],
        "exposure_counts": {},
        "spill_artifact": rel_path,
    }

    return SnapshotResult(
        data=truncated_data,
        artifacts=[artifact],
        truncated=True,
        limits={"max_bytes": max_bytes},
    )
