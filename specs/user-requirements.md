# User requirements (UR)

**Scope note:** This is a single-user system. The workspace maps to one learner profile.

These URs drive the tickets, CLI contract, and executable BDD specs.

## UR-01: On-demand, low-friction sessions
As a learner, I can start a short session at any time and get a small, actionable next step.

- Session length defaults to 7–12 minutes.
- The system is optimized for “stop early, come back tomorrow.”

## UR-02: Natural acquisition & comprehensible input (i+1)
As a learner, I mostly learn through contextual exposure (i+1), not rote rule memorization.

- The engine must store **exposure facts** so the model can keep content near i+1.

## UR-03: Persistent learning state across modalities
As a learner, my learning state persists and can be queried by the model.

The system must track (as facts):
- when an item was first seen / last seen
- exposure counts, grouped by modality (reading/listening/speaking/typing)
- review outcomes (grades, next due times)
- pronunciation attempts and assessments (if enabled)

## UR-04: Spaced repetition primitives
As a learner, I can review due items and record outcomes.

Engine capabilities (facts + mechanical scheduling only):
- list due items (`review start`)
- record grades (`review grade`)
- bury items (`review bury`)
- preview upcoming load (`srs preview`)

## UR-05: Pronunciation practice pipeline (audio in, audio out)
As a learner, I can practice speaking and get structured feedback with artifacts ready to send.

- Input: Telegram voice note (ogg/opus) or wav
- Output: normalized audio, transcript, assessment, feedback voice note
- Audio is implemented via pluggable backends with auditable feature flags (see `specs/audio-backends.md`).

## UR-06: Graded input materials (cached)
As a learner, I can receive stories/dialogues/exercises constrained by my level.

- The model generates content.
- The engine caches and serves it (facts + artifacts only).

## UR-07: Progress facts
As a learner, I can see progress metrics and recent activity as facts (not advice).

Examples:
- exposure counts by week + modality
- review load trends
- pronunciation trend summaries

## UR-08: HSK alignment audits (vocab + grammar; chars optional)
As a learner, I can request HSK coverage/compliance reports with evidence rows.

Requirements:
- report distinguishes **vocabulary** vs **grammar**
- characters are out of scope for v1 (see `specs/hsk-scope.md`)
- report includes bounded evidence rows and can spill to artifacts when needed

## UR-09: ZFC boundary (no recommendations)
As an operator, I can rely on the engine to never produce “what to learn next” recommendations.

- Engine outputs are facts + artifacts, bounded and schema-validated.
- Pedagogical decisions live in the model/Skill.

## UR-10: Deterministic artifact retention
As an operator, I can keep the workspace from growing unboundedly.

- GC is deterministic and safe (`xuezh gc`)
- retention windows are configurable via env vars
