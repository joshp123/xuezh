from __future__ import annotations

import json
import sqlite3
from dataclasses import dataclass
from datetime import datetime, timedelta

from xuezh.core import clock, db, ids


@dataclass(frozen=True)
class DueItem:
    item_id: str
    due_at: str


def _interval_days(rule: str, grade: int) -> int:
    if rule == "leitner":
        table = {0: 1, 1: 1, 2: 2, 3: 4, 4: 7, 5: 14}
    else:
        # sm2-like fixed mapping (mechanical, no adaptive heuristics)
        table = {0: 1, 1: 1, 2: 2, 3: 4, 4: 7, 5: 14}
    return table.get(grade, 1)


def schedule_next_due(*, grade: int, now: datetime, rule: str | None, next_due: str | None) -> tuple[str, str | None]:
    if next_due:
        dt = clock.parse_utc_iso(next_due)
        return dt.isoformat(), None
    applied_rule = rule or "sm2"
    days = _interval_days(applied_rule, grade)
    due_at = now + timedelta(days=days)
    return due_at.isoformat(), applied_rule


def list_due_items(*, limit: int, now: datetime) -> list[DueItem]:
    db_path = db.init_db()
    conn = sqlite3.connect(db_path)
    try:
        rows = conn.execute(
            """
            SELECT item_id, due_at
            FROM user_knowledge
            WHERE due_at IS NOT NULL AND due_at <= ?
            ORDER BY due_at ASC, item_id ASC
            LIMIT ?
            """,
            (now.isoformat(), limit),
        ).fetchall()
        return [DueItem(item_id=row[0], due_at=row[1]) for row in rows]
    finally:
        conn.close()


def record_review_event(
    *,
    item_id: str,
    event_type: str,
    payload: dict,
    now: datetime,
) -> None:
    db_path = db.init_db()
    conn = sqlite3.connect(db_path)
    try:
        event_id = ids.event_id_ulid()
        conn.execute(
            """
            INSERT INTO review_events (id, item_id, event_type, ts, session_id, payload_json)
            VALUES (?, ?, ?, ?, ?, ?)
            """,
            (event_id, item_id, event_type, now.isoformat(), None, json.dumps(payload, ensure_ascii=False)),
        )
        conn.commit()
    finally:
        conn.close()


def upsert_knowledge(
    *,
    item_id: str,
    due_at: str,
    grade: int | None,
    now: datetime,
) -> None:
    db_path = db.init_db()
    conn = sqlite3.connect(db_path)
    try:
        item_type = ids.item_type(item_id) or "unknown"
        row = conn.execute(
            "SELECT seen_count FROM user_knowledge WHERE item_id = ?",
            (item_id,),
        ).fetchone()
        if row:
            seen_count = int(row[0]) + 1
            conn.execute(
                """
                UPDATE user_knowledge
                SET due_at = ?, last_grade = ?, last_seen_at = ?, seen_count = ?
                WHERE item_id = ?
                """,
                (due_at, grade, now.isoformat(), seen_count, item_id),
            )
        else:
            conn.execute(
                """
                INSERT INTO user_knowledge
                (item_id, item_type, modality, first_seen_at, last_seen_at, seen_count, due_at, last_grade)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?)
                """,
                (item_id, item_type, "unknown", now.isoformat(), now.isoformat(), 1, due_at, grade),
            )
        conn.commit()
    finally:
        conn.close()


def preview_due(*, days: int, now: datetime) -> dict[str, int]:
    db_path = db.init_db()
    conn = sqlite3.connect(db_path)
    try:
        rows = conn.execute(
            "SELECT due_at FROM user_knowledge WHERE due_at IS NOT NULL",
        ).fetchall()
    finally:
        conn.close()

    forecast: dict[str, int] = {}
    for (due_at,) in rows:
        due_dt = clock.parse_utc_iso(due_at)
        delta = (due_dt.date() - now.date()).days
        if 0 <= delta <= days:
            key = due_dt.date().isoformat()
            forecast[key] = forecast.get(key, 0) + 1
    return dict(sorted(forecast.items()))
