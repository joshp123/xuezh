from __future__ import annotations

import csv
import hashlib
import json
import sqlite3
from pathlib import Path

from xuezh.core import clock, db, ids


def _sha1_hex(path: Path) -> str:
    h = hashlib.sha1()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(8192), b""):
            h.update(chunk)
    return h.hexdigest()


def _dataset_id(dataset_type: str, version: str) -> str:
    payload = f"{dataset_type}|{version}"
    return f"ds_{hashlib.sha1(payload.encode('utf-8')).hexdigest()[:12]}"


def _read_csv(path: Path) -> list[dict[str, str]]:
    with path.open("r", encoding="utf-8") as handle:
        reader = csv.DictReader(handle)
        return [row for row in reader if any((v or "").strip() for v in row.values())]


def _insert_dataset(conn: sqlite3.Connection, *, dataset_id: str, dataset_type: str, version: str, source: str) -> None:
    conn.execute(
        """
        INSERT OR IGNORE INTO datasets (id, dataset_type, version, source, ingested_at)
        VALUES (?, ?, ?, ?, ?)
        """,
        (dataset_id, dataset_type, version, source, clock.now_utc().isoformat()),
    )


def _insert_dataset_item(
    conn: sqlite3.Connection, *, dataset_id: str, item_id: str, item_type: str, payload: dict
) -> None:
    conn.execute(
        """
        INSERT OR IGNORE INTO dataset_items (dataset_id, item_id, item_type, payload_json)
        VALUES (?, ?, ?, ?)
        """,
        (dataset_id, item_id, item_type, json.dumps(payload, ensure_ascii=False, sort_keys=True)),
    )


def import_dataset(dataset_type: str, path: str) -> tuple[str, int]:
    file_path = Path(path).expanduser()
    if not file_path.exists():
        raise FileNotFoundError(f"Dataset file not found: {file_path}")

    version = _sha1_hex(file_path)
    dataset_id = _dataset_id(dataset_type, version)
    rows = _read_csv(file_path)

    # Frequency datasets must be ordered by rank (ascending). HSK datasets preserve file order.
    if dataset_type == "frequency":
        rows = sorted(rows, key=lambda r: int(r["frequency_rank"]))

    db_path = db.init_db()
    conn = sqlite3.connect(db_path)
    try:
        conn.execute("PRAGMA foreign_keys = ON;")
        _insert_dataset(conn, dataset_id=dataset_id, dataset_type=dataset_type, version=version, source=str(file_path))

        for idx, row in enumerate(rows, start=1):
            if dataset_type == "hsk_vocab":
                level = int(row["hsk_level"])
                hanzi = row["hanzi"]
                pinyin = row["pinyin"]
                meanings = row["meanings"]
                item_id = ids.word_id(hanzi=hanzi, pinyin=pinyin)

                conn.execute(
                    """
                    INSERT OR IGNORE INTO words (id, hanzi, pinyin, definition, source, created_at)
                    VALUES (?, ?, ?, ?, ?, ?)
                    """,
                    (item_id, hanzi, pinyin, meanings, "dataset:hsk_vocab", clock.now_utc().isoformat()),
                )
                payload = {
                    "hsk_level": level,
                    "hanzi": hanzi,
                    "pinyin": pinyin,
                    "meanings": meanings,
                    "order": idx,
                }
                _insert_dataset_item(conn, dataset_id=dataset_id, item_id=item_id, item_type="word", payload=payload)

            elif dataset_type == "hsk_chars":
                level = int(row["hsk_level"])
                character = row["character"]
                pinyin = row["pinyin"]
                meanings = row["meanings"]
                item_id = ids.char_id(character=character)

                conn.execute(
                    """
                    INSERT OR IGNORE INTO characters (id, character, pinyin, definition, source, created_at)
                    VALUES (?, ?, ?, ?, ?, ?)
                    """,
                    (item_id, character, pinyin, meanings, "dataset:hsk_chars", clock.now_utc().isoformat()),
                )
                payload = {
                    "hsk_level": level,
                    "character": character,
                    "pinyin": pinyin,
                    "meanings": meanings,
                    "order": idx,
                    "radical": row.get("radical"),
                    "stroke_count": row.get("stroke_count"),
                }
                _insert_dataset_item(conn, dataset_id=dataset_id, item_id=item_id, item_type="character", payload=payload)

            elif dataset_type == "hsk_grammar":
                level = int(row["hsk_level"])
                grammar_key = row["grammar_id"]
                title = row["title"]
                pattern = row["pattern"]
                examples = row["examples"]
                item_id = ids.grammar_id(grammar_key=grammar_key)

                conn.execute(
                    """
                    INSERT OR IGNORE INTO grammar_points (id, grammar_key, title, notes, source, created_at)
                    VALUES (?, ?, ?, ?, ?, ?)
                    """,
                    (item_id, grammar_key, title, pattern, "dataset:hsk_grammar", clock.now_utc().isoformat()),
                )
                payload = {
                    "hsk_level": level,
                    "grammar_id": grammar_key,
                    "title": title,
                    "pattern": pattern,
                    "examples": examples,
                    "order": idx,
                }
                _insert_dataset_item(conn, dataset_id=dataset_id, item_id=item_id, item_type="grammar", payload=payload)

            elif dataset_type == "frequency":
                rank = int(row["frequency_rank"])
                hanzi = row["hanzi"]
                pinyin = row.get("pinyin", "") or ""
                notes = row.get("notes")
                item_id = ids.word_id(hanzi=hanzi, pinyin=pinyin)

                conn.execute(
                    """
                    INSERT OR IGNORE INTO words (id, hanzi, pinyin, definition, source, created_at)
                    VALUES (?, ?, ?, ?, ?, ?)
                    """,
                    (item_id, hanzi, pinyin, None, "dataset:frequency", clock.now_utc().isoformat()),
                )
                payload = {
                    "frequency_rank": rank,
                    "hanzi": hanzi,
                    "pinyin": pinyin,
                    "notes": notes,
                }
                _insert_dataset_item(conn, dataset_id=dataset_id, item_id=item_id, item_type="word", payload=payload)

            else:
                raise ValueError(f"Unsupported dataset type: {dataset_type}")

        conn.commit()
    finally:
        conn.close()

    return dataset_id, len(rows)
