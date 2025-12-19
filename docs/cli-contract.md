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
- Exit codes: `ok:true` => 0, `ok:false` => non-zero (policy: `specs/errors.md`).
- Time windows use rolling durations in UTC (policy: `specs/time.md`).
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
- `xuezh report hsk --level 1..6|7-9 --window 30d --max-items 200 --max-bytes 200000 [--include-chars] --json`
- command id: `report.hsk`
- output schema: `schemas/report.hsk.schema.json`
- coverage includes `known/unknown` splits; counts by level preserve upstream labels (e.g. `"7-9"`)

### report mastery
- `xuezh report mastery --item-type word|character|grammar --window 90d --max-items 200 --max-bytes 200000 --json`
- command id: `report.mastery`

### report due
- `xuezh report due --limit 50 --max-bytes 200000 --json`
- command id: `report.due`

### audio convert
- `xuezh audio convert --in <path> --out <path> --format wav|ogg|mp3 --backend ffmpeg --json`
- command id: `audio.convert`

### audio tts
- `xuezh audio tts --text "<text>" --voice "<voice>" --out <path> --backend edge-tts --json`
- command id: `audio.tts`

### audio process-voice
- `xuezh audio process-voice --in <voice.ogg> --ref-text "<text>" --json`
- command id: `audio.process-voice`
- output schema: `schemas/audio.process-voice.schema.json`
- default pronunciation backend: `azure.speech` (requires `AZURE_SPEECH_KEY` + `AZURE_SPEECH_REGION`)

### audio backend selection (deterministic)
- Global override: `XUEZH_AUDIO_BACKEND=<backend_id>`
- Per-command overrides:
  - `XUEZH_AUDIO_PROCESS_VOICE_BACKEND=<backend_id>`
  - `XUEZH_AUDIO_CONVERT_BACKEND=<backend_id>`
  - `XUEZH_AUDIO_TTS_BACKEND=<backend_id>`
- CLI flags still apply for `audio convert` / `audio tts` (`--backend`); env vars override defaults.
- Precedence: CLI flag > per-command env > global env > default.

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
