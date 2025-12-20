#!/usr/bin/env python3
from __future__ import annotations

import argparse
import csv
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(REPO_ROOT / "src"))

from xuezh.core import datasets, paths  # noqa: E402

LEVELS_V1 = {"1", "2", "3", "4", "5", "6"}


def _normalize_examples(text: str) -> str:
    if not text:
        return ""
    value = text.replace("\uFF1B", "||").replace(";", "||")
    parts = [part.strip() for part in value.split("||") if part.strip()]
    return "||".join(parts)


def _write_csv(path: Path, headers: list[str], rows: list[dict[str, str]]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8", newline="") as handle:
        writer = csv.DictWriter(handle, fieldnames=headers)
        writer.writeheader()
        writer.writerows(rows)


def _load_vocab(source_path: Path) -> list[dict[str, str]]:
    rows: list[dict[str, str]] = []
    with source_path.open("r", encoding="utf-8", newline="") as handle:
        reader = csv.DictReader(handle)
        for row in reader:
            level = (row.get("Level") or "").strip()
            if level not in LEVELS_V1:
                continue
            hanzi = (row.get("Simplified") or "").strip()
            pinyin = (row.get("Pinyin") or "").strip()
            if not hanzi:
                continue
            rows.append(
                {
                    "hsk_level": level,
                    "hanzi": hanzi,
                    "pinyin": pinyin,
                    "meanings": "",
                    "pos": (row.get("POS") or "").strip(),
                    "notes": (row.get("CEDICT") or "").strip(),
                }
            )
    return rows


def _load_grammar(source_path: Path) -> list[dict[str, str]]:
    rows: list[dict[str, str]] = []
    with source_path.open("r", encoding="utf-8", newline="") as handle:
        reader = csv.DictReader(handle)
        for row in reader:
            level = (row.get("Level") or "").strip()
            if level not in LEVELS_V1:
                continue
            no = (row.get("No") or "").strip()
            if not no:
                continue
            try:
                num = int(no)
            except ValueError:
                continue
            grammar_id = f"HSK{level}-G{num:03d}"
            rows.append(
                {
                    "hsk_level": level,
                    "grammar_id": grammar_id,
                    "title": (row.get("Category") or "").strip(),
                    "pattern": (row.get("Details") or "").strip(),
                    "examples": _normalize_examples((row.get("Content") or "").strip()),
                }
            )
    return rows


def _load_chars(source_path: Path) -> list[dict[str, str]]:
    rows: list[dict[str, str]] = []
    with source_path.open("r", encoding="utf-8", newline="") as handle:
        reader = csv.DictReader(handle)
        for row in reader:
            level = (row.get("Level") or "").strip()
            if level not in LEVELS_V1:
                continue
            character = (row.get("Hanzi") or "").strip()
            if not character:
                continue
            rows.append(
                {
                    "hsk_level": level,
                    "character": character,
                    "pinyin": "",
                    "meanings": "",
                    "radical": "",
                    "stroke_count": "",
                }
            )
    return rows


def _import_dataset(dataset_type: str, path: Path, *, enabled: bool) -> None:
    if not enabled:
        print(f"skip import {dataset_type}: --no-import")
        return
    dataset_id, rows_loaded = datasets.import_dataset(dataset_type, str(path))
    print(f"import {dataset_type}: rows={rows_loaded} dataset_id={dataset_id}")


def main() -> int:
    parser = argparse.ArgumentParser(description="Seed ivankra/hsk30 into xuezh (vocab+grammar, levels 1-6).")
    parser.add_argument("--source", required=True, help="Path to ivankra/hsk30 directory")
    parser.add_argument("--out-dir", help="Where to write canonical CSVs (default: workspace cache)")
    parser.add_argument("--include-chars", action="store_true", help="Also import HSK chars (not v1 default)")
    parser.add_argument("--no-import", action="store_true", help="Only write CSVs; do not import")
    args = parser.parse_args()

    source_dir = Path(args.source).expanduser().resolve()
    vocab_src = source_dir / "hsk30.csv"
    grammar_src = source_dir / "hsk30-grammar.csv"
    chars_src = source_dir / "hsk30-chars.csv"

    missing = [str(p) for p in (vocab_src, grammar_src, chars_src) if not p.exists()]
    if missing:
        print("missing source files:", ", ".join(missing), file=sys.stderr)
        return 2

    if args.out_dir:
        out_dir = Path(args.out_dir).expanduser().resolve()
    else:
        out_dir = paths.ensure_workspace() / "cache" / "hsk30"

    vocab_rows = _load_vocab(vocab_src)
    grammar_rows = _load_grammar(grammar_src)
    chars_rows = _load_chars(chars_src) if args.include_chars else []

    vocab_out = out_dir / "hsk_vocab.csv"
    grammar_out = out_dir / "hsk_grammar.csv"
    chars_out = out_dir / "hsk_chars.csv"

    _write_csv(
        vocab_out,
        headers=["hsk_level", "hanzi", "pinyin", "meanings", "pos", "notes"],
        rows=vocab_rows,
    )
    _write_csv(
        grammar_out,
        headers=["hsk_level", "grammar_id", "title", "pattern", "examples"],
        rows=grammar_rows,
    )
    if args.include_chars:
        _write_csv(
            chars_out,
            headers=["hsk_level", "character", "pinyin", "meanings", "radical", "stroke_count"],
            rows=chars_rows,
        )

    print(f"wrote vocab rows={len(vocab_rows)} -> {vocab_out}")
    print(f"wrote grammar rows={len(grammar_rows)} -> {grammar_out}")
    if args.include_chars:
        print(f"wrote chars rows={len(chars_rows)} -> {chars_out}")

    _import_dataset("hsk_vocab", vocab_out, enabled=not args.no_import)
    _import_dataset("hsk_grammar", grammar_out, enabled=not args.no_import)
    if args.include_chars:
        _import_dataset("hsk_chars", chars_out, enabled=not args.no_import)

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
