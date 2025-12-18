from __future__ import annotations

import json
import sqlite3
from dataclasses import dataclass

from xuezh.core import clock, db, ids, paths


@dataclass(frozen=True)
class Event:
    event_id: str
    event_type: str
    ts: str
    modality: str
    items: list[str]
    context: str | None


def _read_items_file(path: str) -> list[str]:
    resolved = paths.resolve_in_workspace(path)
    items: list[str] = []
    with resolved.open("r", encoding="utf-8") as handle:
        for line in handle:
            item = line.strip()
            if not item:
                continue
            items.append(item)
    return items


def parse_items(*, items: str | None, items_file: str | None) -> list[str]:
    parsed: list[str] = []
    if items:
        parsed.extend([part.strip() for part in items.split(",") if part.strip()])
    if items_file:
        parsed.extend(_read_items_file(items_file))

    for item in parsed:
        if not ids.is_item_id(item):
            raise ValueError(f"Invalid item id: {item}")
    return parsed


def log_event(
    *,
    event_type: str,
    modality: str,
    items: list[str],
    context: str | None,
) -> Event:
    now = clock.now_utc()
    event_id = ids.event_id_ulid()

    db_path = db.init_db()
    conn = sqlite3.connect(db_path)
    try:
        conn.execute(
            """
            INSERT INTO events (id, event_type, ts, modality, items_json, context, payload_json)
            VALUES (?, ?, ?, ?, ?, ?, ?)
            """,
            (
                event_id,
                event_type,
                now.isoformat(),
                modality,
                json.dumps(items, ensure_ascii=False),
                context,
                json.dumps({}, ensure_ascii=False),
            ),
        )
        conn.commit()
    finally:
        conn.close()

    return Event(
        event_id=event_id,
        event_type=event_type,
        ts=now.isoformat(),
        modality=modality,
        items=items,
        context=context,
    )


def list_events(*, since: str, limit: int) -> list[Event]:
    now = clock.now_utc()
    since_dt = now - clock.parse_duration(since)

    db_path = db.init_db()
    conn = sqlite3.connect(db_path)
    try:
        rows = conn.execute(
            """
            SELECT id, event_type, ts, modality, items_json, context
            FROM events
            WHERE ts >= ?
            ORDER BY ts ASC, id ASC
            LIMIT ?
            """,
            (since_dt.isoformat(), limit),
        ).fetchall()
    finally:
        conn.close()

    events: list[Event] = []
    for row in rows:
        items = json.loads(row[4]) if row[4] else []
        events.append(
            Event(
                event_id=row[0],
                event_type=row[1],
                ts=row[2],
                modality=row[3],
                items=items,
                context=row[5],
            )
        )
    return events


def exposure_counts(*, since: str) -> dict[str, int]:
    now = clock.now_utc()
    since_dt = now - clock.parse_duration(since)

    db_path = db.init_db()
    conn = sqlite3.connect(db_path)
    try:
        rows = conn.execute(
            """
            SELECT modality, COUNT(*)
            FROM events
            WHERE event_type = 'exposure' AND ts >= ?
            GROUP BY modality
            """,
            (since_dt.isoformat(),),
        ).fetchall()
    finally:
        conn.close()

    return {row[0]: int(row[1]) for row in rows}
