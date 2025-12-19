# Audio backends (pluggable, deterministic)

Audio commands (`xuezh audio ...`) are implemented via **pluggable backends**.

ZFC/Unix boundary rule:
- The engine never auto-selects a backend via heuristics.
- Backend choice must be **explicit and auditable** (via fixed defaults or explicit flags).

## Backend IDs

Backend IDs are stable strings (opaque identifiers). Examples:
- `ffmpeg` (mechanical audio conversion)
- `edge-tts` (TTS)
- `whisper` (STT)
- `local` (local/dev pronunciation assessment)
- `azure.speech` (paid/online backend; requires secrets)

The contract does not enumerate allowed IDs; it only defines how they are reported.

## Selection

Public commands:
- `audio.convert` and `audio.tts` accept `--backend <BACKEND_ID>`.
- `audio.process-voice` does **not** expose a backend override; it uses a documented default.

Internal primitives (`audio.stt`, `audio.assess`) accept `--backend` but are not part of the public contract.

Environment overrides (deterministic, explicit):
- Global: `XUEZH_AUDIO_BACKEND=<BACKEND_ID>`
- Per-command:
  - `XUEZH_AUDIO_PROCESS_VOICE_BACKEND`
  - `XUEZH_AUDIO_CONVERT_BACKEND`
  - `XUEZH_AUDIO_TTS_BACKEND`

Precedence:
1) CLI flag (when present)
2) Per-command env var
3) Global env var
4) Command default

## Azure Speech (pronunciation assessment)

Backend id: `azure.speech`
Default for `audio.process-voice`.

Requirements:
- `AZURE_SPEECH_KEY`
- `AZURE_SPEECH_REGION`

Free-tier policy:
- Use the free-tier quota only; if Azure returns quota/limit errors, the CLI surfaces a typed error.

## Deterministic features (auditable)

When implemented, audio commands include backend metadata in `data.backend`:
- `id`: the backend ID actually used
- `features`: a deterministic list of capability flags (strings)

This is used for auditability and to keep the Skill’s behavior deterministic.

## CI policy (YOLO)

For now, audio command tests are expected to run against **real tools/backends** (ffmpeg, edge-tts, whisper, etc.)
once those commands are implemented.

If CI becomes flaky due to network/tool variance, revisit this policy and split out “real backend” checks into a
separate workflow or add deterministic local backends for CI.
