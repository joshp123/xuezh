# CLI contract (authoritative)

This document is the **single source of truth** for how the LLM Skill should call the `xuezh` engine.

**Do not copy/paste or re-describe commands in multiple places**.
- The Skill should link here.
- BDD scenarios should use the commands defined here.
- JSON Schemas in `schemas/` must match these outputs.
- Tests must validate actual CLI outputs against those schemas.

## Global invariants
- ID scheme is authoritative: `specs/id-scheme.md`.

- Single-user system (no multi-user semantics).
- Every command supports `--json` and returns a JSON envelope.
- No command ever returns recommendations (ZFC): no `recommended_next`, no `priority_score`, etc.
- Outputs are bounded by `--limit` and/or `--max-bytes`. When exceeding bounds, spill to an artifact file and return a handle/path.

## Envelope
All commands return one of:
- OK envelope (`schemas/envelope.ok.schema.json`)
- ERR envelope (`schemas/envelope.err.schema.json`)

## Commands

### version
- `xuezh version [--json]`
- command id: `version`

### snapshot
- `xuezh snapshot --window 30d --due-limit 80 --evidence-limit 200 --max-bytes 200000 --json`
- command id: `snapshot`
- output schema: `schemas/snapshot.schema.json`

### db init
- `xuezh db init --json`
- command id: `db.init`

### dataset import
- `xuezh dataset import --type <hsk_vocab|hsk_chars|hsk_grammar|frequency> --path <file> --json`
- command id: `dataset.import`
- dataset format: see `specs/datasets-format.md`
- provenance/licensing: see `specs/datasets/provenance.md`

### review start
- `xuezh review start --limit 10 --json`
- command id: `review.start`
- output schema: `schemas/review.start.schema.json`

### review grade
- `xuezh review grade --item <ITEM_ID> --grade 0..5 [--next-due <ISO>] [--rule sm2|leitner] --json`
- command id: `review.grade`
- output schema: `schemas/review.grade.schema.json`

### review bury
- `xuezh review bury --item <ITEM_ID> [--reason ...] --json`
- command id: `review.bury`

### srs preview
- `xuezh srs preview --days 14 --json`
- command id: `srs.preview`

### report hsk
- `xuezh report hsk --level 1..6 --window 30d --max-items 200 --max-bytes 200000 [--include-chars] --json`
- command id: `report.hsk`
- output schema: `schemas/report.hsk.schema.json`

### report mastery
- `xuezh report mastery --item-type word|character|grammar --window 90d --max-items 200 --max-bytes 200000 --json`
- command id: `report.mastery`

### report due
- `xuezh report due --limit 50 --max-bytes 200000 --json`
- command id: `report.due`

### audio convert
- `xuezh audio convert --in <path> --out <path> --format wav|ogg|mp3 --json`
- command id: `audio.convert`

### audio tts
- `xuezh audio tts --text "<text>" --voice "<voice>" --out <path> --json`
- command id: `audio.tts`

### audio stt
- `xuezh audio stt --in <path> --json`
- command id: `audio.stt`

### audio assess
- `xuezh audio assess --ref-text "<text>" --in <path> --mode local|azure --json`
- command id: `audio.assess`

### audio process-voice
- `xuezh audio process-voice --in <voice.ogg> --ref-text "<text>" --mode local|azure --json`
- command id: `audio.process-voice`
- output schema: `schemas/audio.process-voice.schema.json`

### content cache put/get
- `xuezh content cache put --type story|dialogue|exercise --key <hash> --in <file> --json`
- `xuezh content cache get --type story|dialogue|exercise --key <hash> --json`
- command ids:
  - `content.cache.put`
  - `content.cache.get`

### doctor
- `xuezh doctor --json`
- command id: `doctor`


### event log
- `xuezh event log --type <exposure|review|pronunciation_attempt|content_served> --modality <reading|listening|speaking|typing|mixed> [--items <comma_list>] [--items-file <path>] [--context <str>] --json`
- command id: `event.log`
- output schema: `schemas/event.log.schema.json`

### event list
- `xuezh event list --since 7d --limit 200 --json`
- command id: `event.list`
- output schema: `schemas/event.list.schema.json`

### gc
- `xuezh gc --dry-run --json`
- `xuezh gc --apply --json`
- command id: `gc`
- output schema: `schemas/gc.schema.json`
