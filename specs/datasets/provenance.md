# Dataset provenance & licensing notes (important)

**Goal:** make HSK/frequency data reproducible and legally unambiguous.

This repo does **not** bundle full HSK datasets by default.
It provides:
- import formats (`specs/datasets-format.md`)
- tiny test fixtures under `tests/fixtures/datasets/` (minimal samples)

For real datasets, choose an open-licensed source and record its version/commit hash.

## Recommended open sources (examples)

1) Open Language Profiles — Mandarin Chinese dataset (Zero to Hero)
   - Repo: https://github.com/openlanguageprofiles/olp-zh-zerotohero
   - Terms:
     - CC-CEDICT: CC BY-SA 4.0
     - the rest: CC0 / public domain
   - Includes: vocab list with HSK levels + grammar list

2) ivankra/hsk30 (HSK 3.0 vocab + grammar + chars)
   - Repo: https://github.com/ivankra/hsk30
   - License: MIT
   - Includes: csv files, grammar list, chars list

3) elkmovie/hsk30 (HSK 3.0 word + char lists)
   - Repo: https://github.com/elkmovie/hsk30
   - License: MIT
   - Notes: extracted/OCR from official PDF

4) clem109/hsk-vocabulary (HSK vocab with examples)
   - Repo: https://github.com/clem109/hsk-vocabulary
   - License: MIT

## Caution: “official HSK lists”
Many sites host official PDFs/spreadsheets, but licensing/redistribution rights may be unclear.
Prefer sources with explicit open licenses.

## Required recording for imports
When importing any dataset, record:
- source URL
- license
- commit hash or release tag
- import date

Ticket T-04 requires making this provenance record a first-class artifact in the workspace.

## Scope
HSK audits in v1 focus on vocab+grammar; chars are optional (see `specs/hsk-scope.md`).
