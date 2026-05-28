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

### learner state
- `xuezh learner state --json`
- command id: `learner.state`
- output schema: `schemas/learner.state.schema.json`
- returns the full canonical cram deck as compact columnar JSON for LLM context; `data.columns` names the fields and every row in `data.cards` follows that order
- includes canonical card text, category context, live Pleco-style score facts, learned/due booleans, and review history; it intentionally omits pinyin, audio paths, source indices, and recommendations
- `data.state_hash` changes when visible learner state changes, so callers can cache the payload and reload only when stale

### db init
- `xuezh db init --json`
- command id: `db.init`

### dataset import
- `xuezh dataset import --type <hsk_vocab|hsk_chars|hsk_grammar|frequency> --path <file> --json`
- command id: `dataset.import`
- dataset format: see `specs/datasets-format.md`
- provenance/licensing: see `specs/datasets/provenance.md`

### hellochinese import
- `xuezh hellochinese import --path <Full Pleco Import.txt> [--audio none|sentence] [--voices <comma_list>] --json`
- command id: `hellochinese.import`
- output schema: `schemas/hellochinese.import.schema.json`
- imports one review item per canonical Pleco-text row; section headers become categories and file order is learning order
- source sentence text is cleaned before storage/audio/card display; raw text is retained only as provenance
- `--audio sentence` uses the existing `audio tts` backend and should only be used after a one-sentence smoke test succeeds in `devenv`

### hellochinese audio-backfill
- `xuezh hellochinese audio-backfill [--limit N] [--concurrency N] [--voices <comma_list>] --json`
- command id: `hellochinese.audio-backfill`
- output schema: `schemas/hellochinese.audio-backfill.schema.json`
- generates missing sentence audio for already-imported HelloChinese items
- audio generation may run concurrently; SQLite updates are serialized so per-voice paths are not lost

### travel import
- `xuezh travel import --path <Travel Survival Pleco Import.txt> [--audio none|sentence] [--voices <comma_list>] --json`
- command id: `travel.import`
- output schema: `schemas/travel.import.schema.json`
- imports one review item per canonical Pleco-text row; `Travel Survival/` is stripped from stored category names

### pleco score-import
- `xuezh pleco score-import --path <Pleco Flash Backup.pqb> --json`
- command id: `pleco.score-import`
- output schema: `schemas/pleco.score-import.schema.json`
- imports only score/recency metadata from the Pleco backup; it does not read Pleco card text as canonical content
- maps Pleco scores by root category, child category, and assignment order; count mismatches are left unseeded and reported

### cram overview
- `xuezh cram overview --json`
- command id: `cram.overview`
- output schema: `schemas/cram.overview.schema.json`
- returns source/category facts only; product UI derives practice pools from live score rows

### cram audio-backfill
- `xuezh cram audio-backfill [--source all|hellochinese|travel_survival] [--limit N] [--concurrency N] [--voices <comma_list>] [--rates <voice=rate,...>] [--replace] --json`
- command id: `cram.audio-backfill`
- output schema: `schemas/cram.audio-backfill.schema.json`
- generates missing sentence audio for the canonical cram deck
- default voices are the four calibrated Mandarin voices, with slower per-voice Edge TTS rates; `--replace` clears stored audio paths before regenerating the selected scope

### cram next
- `xuezh cram next --limit 1 --json`
- command id: `cram.next`
- output schema: `schemas/cram.next.schema.json`
- returns not-learned or due cards first, with missed-before cards and lower scores earlier inside that pool
- card front is the Chinese sentence with the target word available for highlighting by clients

### cram grade
- `xuezh cram grade --item <ITEM_ID> --grade incorrect|correct --json`
- command id: `cram.grade`
- output schema: `schemas/cram.grade.schema.json`
- updates the live Pleco-style score facts for the card:
  - `incorrect`: answer quality 2; resets score to the imported profile's incorrect score
  - `correct`: answer quality 6; increases score using the imported profile settings
- `next_due_at` is `last_reviewed + score / points_per_day`

### review start
- `xuezh review start --limit 10 --json`
- command id: `review.start`
- output schema: `schemas/review.start.schema.json`
- output includes separate `recall_items` and `pronunciation_items` (with `items` as recall alias)

### review grade
- `xuezh review grade --item <ITEM_ID> --recall 0..5 [--pronunciation 0..5] [--next-due <ISO>] [--rule sm2|leitner] --json`
- `--grade` remains as a recall-only alias for backward compatibility
- command id: `review.grade`
- output schema: `schemas/review.grade.schema.json`

### review bury
- `xuezh review bury --item <ITEM_ID> [--reason ...] --json`
- command id: `review.bury`

### srs preview
- `xuezh srs preview --days 14 --json`
- command id: `srs.preview`
- output includes separate `forecast.recall` and `forecast.pronunciation`

### report hsk
- `xuezh report hsk --level 1..6|7-9 --window 30d --max-items 200 --max-bytes 200000 --json`
- command id: `report.hsk`
- output schema: `schemas/report.hsk.schema.json`
- coverage includes `known/unknown` splits per level (vocab + grammar only)
- `--level 7-9` selects the upstream bucket; numeric levels include the bucket when `--level >= 7`

### report mastery
- `xuezh report mastery --item-type word|character|grammar --window 90d --max-items 200 --max-bytes 200000 --json`
- command id: `report.mastery`

### report due
- `xuezh report due --limit 50 --max-bytes 200000 --json`
- command id: `report.due`

### audio convert
- `xuezh audio convert --in <path> --out <path> --format wav|ogg|mp3 --backend ffmpeg --json`
- command id: `audio.convert`
- `--out` must be inside the workspace (use a relative path like `artifacts/converted.wav`)

### audio tts
- `xuezh audio tts --text "<text>" --voice "<voice>" --out <path> --backend edge-tts --json`
- command id: `audio.tts`
- local mode: `--out` must be inside the workspace (use a relative path like `artifacts/tts.ogg`)
- client-backed mode: `--out` is a local delivery path for the caller; the server still writes the canonical artifact

### audio process-voice
- `xuezh audio process-voice --in <voice.ogg> --ref-text "<text>" --json`
- command id: `audio.process-voice`
- output schema: `schemas/audio.process-voice.schema.json`
- default pronunciation backend: `azure.speech` (configured under `[audio]` and `[azure.speech]`)
- output includes inline `assessment` + `transcript` for actionable feedback; full detail remains in artifacts
- if inline word/phoneme detail is too large, only summary is returned inline and `truncated=true` with full detail in artifacts

### audio backend selection (deterministic)
- Config file: `/etc/xuezh/config.toml` on managed hosts, otherwise `~/.config/xuezh/config.toml`
  - `[audio] process_voice_backend`, `convert_backend`, `tts_backend`
- CLI flags still apply for `audio convert` / `audio tts` (`--backend`).
- Precedence: CLI flag when present, then config file, then command default.
- xuezh-specific env vars are not config inputs.

### client-backed mode
- `[client].server_url = "https://chinese.jjpcodes.com"` makes the CLI call the remote xuezh service for OpenClaw learning workflows.
- `[client]` and `[workspace]` are mutually exclusive.
- In client-backed mode, server artifact paths are audit metadata, not Mac-local filesystem paths.
- Unsupported server-local commands return `UNSUPPORTED_CLIENT_COMMAND` before opening a local workspace.

### config file (optional)
Example:
```
[workspace]
dir = "/var/lib/xuezh"

[audio]
process_voice_backend = "azure.speech"
convert_backend = "ffmpeg"
tts_backend = "edge-tts"
inline_max_bytes = 200000

[azure.speech]
key_file = "/run/agenix/xuezh-azure-speech-key"
region = "westeurope"
```

### content cache put/get
- `xuezh content cache put --type story|dialogue|exercise --key <hash> --in <file> --json`
- `xuezh content cache get --type story|dialogue|exercise --key <hash> --json`
- command ids:
  - `content.cache.put`
  - `content.cache.get`

### doctor
- `xuezh doctor --json`
- command id: `doctor`
- local mode reports workspace, DB, tool, and Azure config checks.
- client-backed mode reports `client.mode`, `server.reachable`, and server-side checks from the remote xuezh service.


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
