# Architecture (ZFC / Unix)

## Components

### 1) Telegram bot + LLM orchestrator (NOT in this repo)
- Chooses lesson plan
- Formats Telegram messages
- Calls this engine via CLI

### 2) `chlearn` engine (THIS repo)
- SQLite persistence + migrations
- Mechanical SRS primitives (due, preview, record)
- Bounded reports (HSK audit, mastery facts)
- Audio pipeline wrappers (convert/tts/stt/assess)
- Content cache (store/retrieve generated content)

## Boundary rules
- Engine never outputs "recommended next".
- Engine never computes priority scores or chooses what to teach.
- Engine may return candidate sets deterministically:
  - due items ordered by due_at
  - unseen HSK items ordered by dataset order/frequency rank
  - evidence rows ordered by stable keys

## CLI JSON envelope
All commands must return either:
- ok envelope: {ok:true, data:{...}, artifacts:[...], ...}
- err envelope: {ok:false, error:{type,message,details}}

See `src/chlearn/core/envelope.py`.

## Single-user scope
This engine is single-user; the workspace maps to one learner profile.

## Artifact retention
See `specs/artifacts/retention.md`.

## Events / exposures
See `specs/events.md`.
