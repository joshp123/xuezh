# Dataset import formats (v0)

The engine must support importing HSK and frequency datasets to enable HSK audits.
This file specifies **the on-disk interchange format** the CLI will accept.

## General rules
- Accept UTF-8 encoded files.
- Prefer CSV with header row.
- All imports are idempotent: importing the same dataset version twice must not duplicate items.
- Ordering:
  - HSK datasets preserve **file order** as dataset order.
  - Frequency datasets preserve `frequency_rank` ordering (ascending).

## 1) HSK vocabulary (`--type hsk_vocab`)
CSV columns (required):
- `hsk_level` (string or int: `1`–`6` or `7-9`)
- `hanzi` (string)
- `pinyin` (string, tone marks preferred)
- `meanings` (string; multiple meanings separated by `|`)

Optional:
- `pos` (string)
- `notes` (string)

## 2) HSK characters (`--type hsk_chars`) (optional for v1)
CSV columns (required):
- `hsk_level` (string or int: `1`–`6` or `7-9`)
- `character` (single Han character)
- `pinyin` (string; if multiple, separate with `|`)
- `meanings` (string; separate with `|`)

Optional:
- `radical`
- `stroke_count` (int)

## 3) HSK grammar points (`--type hsk_grammar`)
CSV columns (required):
- `hsk_level` (string or int: `1`–`6` or `7-9`)
- `grammar_id` (string; stable ID like `HSK3-G012`)
- `title` (string)
- `pattern` (string; e.g., `因为…所以…`)
- `examples` (string; separate examples with `||`)

## 4) Frequency list (`--type frequency`)
CSV columns (required):
- `frequency_rank` (int; 1 is most frequent)
- `hanzi` (string; word)
Optional:
- `pinyin`
- `notes`

## Fixtures
See `tests/fixtures/datasets/*.csv` for minimal example files used in tests.

## Provenance and licensing
See `specs/datasets/provenance.md`.
