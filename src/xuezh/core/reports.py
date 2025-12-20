from __future__ import annotations

import json
import sqlite3
from dataclasses import dataclass
from datetime import datetime
from typing import Any, cast
from pathlib import Path

from xuezh.core import clock, db, jsonio, paths
from xuezh.core.envelope import Artifact


@dataclass(frozen=True)
class ReportResult:
    data: dict[str, Any]
    artifacts: list[Artifact]
    truncated: bool
    limits: dict[str, Any]


def _artifact_path(prefix: str, now: datetime) -> Path:
    root = paths.ensure_workspace()
    day_path = root / "artifacts" / now.strftime("%Y") / now.strftime("%m") / now.strftime("%d")
    day_path.mkdir(parents=True, exist_ok=True)
    filename = f"{prefix}-{now.strftime('%Y%m%dT%H%M%SZ')}.json"
    return day_path / filename


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


def _known_items(conn: sqlite3.Connection, item_type: str | None) -> set[str]:
    if item_type:
        rows = conn.execute(
            "SELECT item_id FROM user_knowledge WHERE item_type = ? AND seen_count > 0",
            (item_type,),
        ).fetchall()
    else:
        rows = conn.execute("SELECT item_id FROM user_knowledge WHERE seen_count > 0").fetchall()
    return {row[0] for row in rows}


def _counts_by_level_stats(items: list[tuple[str, int]], known_ids: set[str]) -> dict[str, dict]:
    by_level: dict[str, dict] = {}
    levels = sorted({level for _, level in items})
    for level in levels:
        level_key = str(level)
        total = sum(1 for _, item_level in items if item_level == level)
        known = sum(1 for item_id, item_level in items if item_level == level and item_id in known_ids)
        unknown = total - known
        coverage_pct = (known / total) if total > 0 else 0.0
        by_level[level_key] = {
            "total": total,
            "known": known,
            "unknown": unknown,
            "coverage_pct": coverage_pct,
        }
    return by_level


def _evidence_rows(
    items: list[tuple[str, int]],
    known_ids: set[str],
    *,
    max_items: int,
) -> list[dict]:
    rows: list[dict] = []
    for item_id, level in sorted(items, key=lambda it: (it[1], it[0])):
        status = "known" if item_id in known_ids else "unknown"
        if status == "unknown":
            rows.append({"item_id": item_id, "level": level, "status": status})
        if len(rows) >= max_items:
            break
    return rows


def _spill_if_needed(
    *, envelope: dict[str, Any], max_bytes: int, prefix: str
) -> tuple[dict[str, Any], list[Artifact], bool]:
    payload = jsonio.dumps(envelope)
    if len(payload.encode("utf-8")) <= max_bytes:
        return envelope, [], False

    now = clock.now_utc()
    spill_path = _artifact_path(prefix, now)
    spill_path.write_text(payload, encoding="utf-8")
    workspace = paths.ensure_workspace()
    rel_path = str(spill_path.relative_to(workspace))
    artifact = Artifact(path=rel_path, mime="application/json", purpose=f"{prefix}_spill", bytes=spill_path.stat().st_size)

    truncated = dict(envelope)
    data = cast(dict[str, Any], envelope.get("data", {}))
    truncated["data"] = {
        "spill_artifact": rel_path,
        "window": data.get("window"),
        "level": data.get("level"),
        "item_type": data.get("item_type"),
    }
    truncated["artifacts"] = [artifact.__dict__]
    truncated["truncated"] = True
    return truncated, [artifact], True


def build_hsk_report(
    *,
    level: str,
    window: str,
    max_items: int,
    max_bytes: int,
    include_chars: bool,
) -> ReportResult:
    db_path = db.init_db()
    conn = sqlite3.connect(db_path)
    try:
        vocab_id = _latest_dataset_id(conn, "hsk_vocab")
        grammar_id = _latest_dataset_id(conn, "hsk_grammar")
        chars_id = _latest_dataset_id(conn, "hsk_chars") if include_chars else None

        vocab_items: list[tuple[str, int]] = []
        grammar_items: list[tuple[str, int]] = []
        chars_items: list[tuple[str, int]] = []

        if vocab_id:
            for item_id, payload_json in conn.execute(
                "SELECT item_id, payload_json FROM dataset_items WHERE dataset_id = ?",
                (vocab_id,),
            ):
                payload = json.loads(payload_json)
                vocab_items.append((item_id, int(payload.get("hsk_level"))))
        if grammar_id:
            for item_id, payload_json in conn.execute(
                "SELECT item_id, payload_json FROM dataset_items WHERE dataset_id = ?",
                (grammar_id,),
            ):
                payload = json.loads(payload_json)
                grammar_items.append((item_id, int(payload.get("hsk_level"))))
        if chars_id:
            for item_id, payload_json in conn.execute(
                "SELECT item_id, payload_json FROM dataset_items WHERE dataset_id = ?",
                (chars_id,),
            ):
                payload = json.loads(payload_json)
                chars_items.append((item_id, int(payload.get("hsk_level"))))

        known_vocab = _known_items(conn, "word")
        known_grammar = _known_items(conn, "grammar")
        known_chars = _known_items(conn, "character")

        vocab_levels = _counts_by_level_stats(vocab_items, known_vocab)
        grammar_levels = _counts_by_level_stats(grammar_items, known_grammar)
        chars_levels = _counts_by_level_stats(chars_items, known_chars)

        def filter_levels(items: list[tuple[str, int]]) -> list[tuple[str, int]]:
            if "-" in level:
                min_level = int(level.split("-")[0])
                return [it for it in items if it[1] >= min_level]
            return [it for it in items if it[1] <= int(level)]

        vocab_items = filter_levels(vocab_items)
        grammar_items = filter_levels(grammar_items)
        chars_items = filter_levels(chars_items)

        coverage: dict[str, dict[str, Any]] = {
            "vocab": {
                "total": len(vocab_items),
                "known": sum(1 for item_id, _ in vocab_items if item_id in known_vocab),
            },
            "grammar": {
                "total": len(grammar_items),
                "known": sum(1 for item_id, _ in grammar_items if item_id in known_grammar),
            },
        }
        coverage["vocab"]["unknown"] = coverage["vocab"]["total"] - coverage["vocab"]["known"]
        coverage["vocab"]["coverage_pct"] = (
            coverage["vocab"]["known"] / coverage["vocab"]["total"]
            if coverage["vocab"]["total"] > 0
            else 0.0
        )
        coverage["grammar"]["unknown"] = coverage["grammar"]["total"] - coverage["grammar"]["known"]
        coverage["grammar"]["coverage_pct"] = (
            coverage["grammar"]["known"] / coverage["grammar"]["total"]
            if coverage["grammar"]["total"] > 0
            else 0.0
        )
        if include_chars:
            coverage["chars"] = {
                "total": len(chars_items),
                "known": sum(1 for item_id, _ in chars_items if item_id in known_chars),
            }
            coverage["chars"]["unknown"] = coverage["chars"]["total"] - coverage["chars"]["known"]
            coverage["chars"]["coverage_pct"] = (
                coverage["chars"]["known"] / coverage["chars"]["total"]
                if coverage["chars"]["total"] > 0
                else 0.0
            )

        data: dict[str, Any] = {
            "level": level,
            "window": window,
            "coverage": coverage,
            "evidence": _evidence_rows(vocab_items, known_vocab, max_items=max_items),
            "counts_by_level": {
                "vocab": vocab_levels,
                "grammar": grammar_levels,
            },
        }
        if include_chars:
            data["counts_by_level"]["chars"] = chars_levels

        envelope = {
            "ok": True,
            "schema_version": "1",
            "command": "report.hsk",
            "data": data,
            "artifacts": [],
            "truncated": False,
            "limits": {"max_items": max_items, "max_bytes": max_bytes},
        }
    finally:
        conn.close()

    envelope, artifacts, truncated = _spill_if_needed(
        envelope=envelope, max_bytes=max_bytes, prefix="report-hsk"
    )
    return ReportResult(
        data=cast(dict[str, Any], envelope["data"]),
        artifacts=artifacts,
        truncated=truncated,
        limits=cast(dict[str, Any], envelope["limits"]),
    )


def build_mastery_report(
    *,
    item_type: str,
    window: str,
    max_items: int,
    max_bytes: int,
) -> ReportResult:
    now = clock.now_utc()
    since = now - clock.parse_duration(window)

    db_path = db.init_db()
    conn = sqlite3.connect(db_path)
    try:
        rows = conn.execute(
            """
            SELECT
              item_id,
              last_seen_at,
              seen_count,
              COALESCE(recall_due_at, due_at),
              COALESCE(recall_last_grade, last_grade),
              pronunciation_due_at,
              pronunciation_last_grade
            FROM user_knowledge
            WHERE item_type = ? AND last_seen_at IS NOT NULL AND last_seen_at >= ?
            ORDER BY last_seen_at DESC, item_id ASC
            """,
            (item_type, since.isoformat()),
        ).fetchall()
    finally:
        conn.close()

    items = [
        {
            "item_id": row[0],
            "last_seen": row[1],
            "times_seen": row[2],
            "recall_due_at": row[3],
            "recall_last_grade": row[4],
            "pronunciation_due_at": row[5],
            "pronunciation_last_grade": row[6],
        }
        for row in rows[:max_items]
    ]

    data: dict[str, Any] = {"item_type": item_type, "window": window, "items": items}
    envelope: dict[str, Any] = {
        "ok": True,
        "schema_version": "1",
        "command": "report.mastery",
        "data": data,
        "artifacts": [],
        "truncated": False,
        "limits": {"max_items": max_items, "max_bytes": max_bytes},
    }

    envelope, artifacts, truncated = _spill_if_needed(
        envelope=envelope, max_bytes=max_bytes, prefix="report-mastery"
    )
    return ReportResult(
        data=cast(dict[str, Any], envelope["data"]),
        artifacts=artifacts,
        truncated=truncated,
        limits=cast(dict[str, Any], envelope["limits"]),
    )
