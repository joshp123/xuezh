# Audio backends (pluggable, deterministic)

Audio commands (`xuezh audio ...`) are implemented via **pluggable backends**.

ZFC/Unix boundary rule:
- The engine never auto-selects a backend via heuristics.
- Backend choice must be **explicit and auditable**.

## Backend IDs

Backend IDs are stable strings (opaque identifiers). Examples:
- `ffmpeg` (mechanical audio conversion)
- `edge-tts` (TTS)
- `whisper` (STT)
- `local` (local/dev pronunciation assessment)
- `azure.speech` (paid/online backend; requires secrets)

The contract does not enumerate allowed IDs; it only defines how they are passed and reported.

## Selection

Every `xuezh audio <cmd>` supports:
- `--backend <BACKEND_ID>`

If omitted, each command uses its documented default backend.

## Deterministic features (auditable)

When implemented, audio commands include backend metadata in `data.backend`:
- `id`: the backend ID actually used
- `features`: a deterministic list of capability flags (strings)

This is used for auditability and to keep the Skill’s behavior deterministic.
