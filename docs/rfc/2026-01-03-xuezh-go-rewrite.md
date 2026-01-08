# RFC: xuezh Go rewrite (drop-in CLI parity)

- Date: 2026-01-03
- Status: Draft
- Audience: CTO, core engine maintainers, ops

## 1) Narrative: what we are building and why

We are rewriting xuezh from Python to Go as a drop-in CLI replacement with
semantic equivalence and full contract parity. The goal is to make the core
engine more robust and predictable while preserving existing workflows and
interfaces so downstream callers (LLM orchestrator, Telegram bot, scripts) do
not change. This aligns with the xuezh North Stars: a thin deterministic shell
(NS-3), auditable instrumentation (NS-2), and low-effort learning (NS-1).

We will keep the Python implementation as the reference until parity is proven
by the existing contract and BDD test suite. The Go binary will be introduced
side-by-side and will take over the `xuezh` command only after tests are green.

## Status hygiene (required)

- When work starts, update Status to Implementing.
- When complete, update Status to Done.
- Update `docs/rfc/README.md` in the same commit.

## 1.1) Non-negotiables

- CLI contract is authoritative (`docs/cli-contract.md`, `specs/cli/contract.json`, `schemas/`).
- Semantic equivalence for all CLI commands in `specs/cli/contract.json`.
- Parity rules for non-deterministic fields are explicit and enforced (see §12).
- SQLite database stays the same (schema, migrations, and on-disk location).
- ZFC/Unix boundary rules remain enforced; no ranking/scoring or planning logic in engine.
- Single-user system only; no `--user` flags.
- Audio backends are explicit and auditable (env or flags only), no auto-selection.
- Darwin-only is acceptable initially; keep implementation simple.

## 2) Goals / Non-goals

Goals:
- Implement all CLI commands with semantic equivalence.
- Preserve JSON envelope, output schemas, and BDD test expectations.
- Keep database paths and migrations identical.
- Provide a side-by-side Go binary (`xuezh-go`) until parity is proven.
- Use existing audio tooling (ffmpeg, edge-tts, whisper, azure speech) via thin pipes.

Non-goals:
- Changing CLI UX, adding new commands, or breaking schemas.
- Replacing SQLite with another datastore.
- Adding scheduling, ranking, or recommendation logic.
- Multi-platform support beyond darwin in phase 1.

## 3) System overview

The Go rewrite is a CLI that reads/writes the same SQLite database and filesystem
artifacts, implements the same contract, and shells out to the same audio tools
or their thin Go wrappers. It will live alongside the Python CLI until parity is
proven by tests and a parity harness comparing outputs.

## 4) Components and responsibilities

- CLI layer (Go): parse args, validate, route to command handlers, emit JSON envelope.
- Core services (Go): datasets, reviews, reports, events, audio, content cache.
- Storage: SQLite schema + migrations unchanged; workspace directory unchanged.
- Tool adapters: ffmpeg, edge-tts, whisper, azure speech (shell-out or thin Go wrappers).
- Parity harness: runs Python and Go CLIs against the same inputs and compares outputs.
  - Canonicalizes output fields per §12 and reports diffs.
  - Proposed location: `tests/parity/` with a CLI runner script in `scripts/parity.sh`.

## 5) Inputs / workflow profiles

Minimum inputs:
- CLI arguments and flags per contract.
- Workspace path (defaults to `~/.clawdbot/workspace/xuezh/`).
- Optional environment variables for audio backend selection and secrets.

Validation rules:
- Enforce schema-valid JSON output for all commands.
- Enforce ID formats per `specs/id-scheme.md`.
- Reject paths that escape the workspace boundary.

## 6) Artifacts / outputs

- JSON envelope outputs matching `schemas/*.schema.json`.
- SQLite database files and backups in the existing workspace layout.
- Audio artifacts (wav/ogg/mp3) as specified by CLI arguments and specs.
- Content cache artifacts (story/dialogue/exercise) as files under workspace.

## 7) State machine (if applicable)

N/A. Commands are request/response operations; state is stored in SQLite and
workspace artifacts.

## 8) API surface (protobuf)

N/A. The system is CLI-based; no protobuf APIs are introduced.

## 9) Interaction model

Users (and upstream orchestration) call `xuezh` with CLI arguments and receive
JSON envelopes on stdout. The Go implementation preserves this contract. During
migration, `xuezh-go` will be used for parity tests and manual verification.

## 10) System interaction diagram

```
Caller -> CLI args -> Go CLI router -> Command handler
      -> SQLite read/write
      -> Tool adapter (ffmpeg/edge-tts/whisper/azure) if needed
      -> JSON envelope output
```

## 11) API call flow (detailed)

Example: `xuezh db init --json`
1) Parse args and flags.
2) Ensure workspace directory exists.
3) Apply migrations if needed.
4) Emit JSON envelope with db path.

Example: `xuezh audio process-voice --in <path> --ref-text <text>`
1) Parse args and flags; resolve backend from flag/env/default.
2) Validate inputs and workspace path.
3) Run ffmpeg normalize step.
4) Run whisper STT (shell-out) and azure/local assess (shell-out SDK or wrapper).
5) Write artifacts and emit JSON envelope referencing them.

Gating rules:
- Fail fast with typed errors if required tools or env vars are missing.
- Emit JSON envelope errors consistent with Python behavior.
- Audio backend resolution uses the exact precedence documented in `specs/audio-backends.md`.

## 12) Determinism and validation

- All command outputs must validate against existing JSON schemas.
- Backend selection is deterministic via flag/env precedence rules.
- Workspace path resolution and path traversal prevention must match Python.
- Parity harness compares Python vs Go outputs for semantic equivalence.
  Allowed tolerances (must be implemented in the harness and documented in diffs):
  - Timestamps: compare by RFC3339 parse; allow <= 2s drift when generated at runtime.
  - ULIDs/IDs: must match when provided as inputs; generated IDs must be structurally valid
    but may differ if not specified by test inputs.
  - Ordering: arrays must be deterministically sorted where Python already sorts;
    otherwise order-insensitive comparison is allowed only for sets explicitly
    defined as unordered in schemas or BDD expectations.
  - Floating point scores (if any): compare with absolute tolerance <= 1e-6.
  - Artifact paths: compare workspace-relative paths; allow differing absolute roots.

## 13) Outputs and materialization

Primary outputs are JSON envelopes on stdout and artifacts in the workspace.
No new formats are introduced. Any later materialization (e.g., report export)
follows the existing contract and schemas.

## 14) Testing philosophy and trust

- Unit tests for Go core logic and adapters.
- Contract tests: `tests/contract/` must pass unchanged.
- BDD tests: `specs/bdd/` scenarios must pass unchanged.
- Parity harness: run both CLIs on the same inputs and compare outputs with
  canonicalization rules in §12.
  - Proposed invocation: `./scripts/parity.sh` (runs Python CLI and Go CLI on
    the same fixtures, compares normalized JSON and artifacts).
- Trust gate: Go CLI is not eligible to replace Python until:
  - contract + BDD tests are green,
  - parity diff set is empty or explicitly approved,
  - and smoke test examples are recorded in the RFC or ticket notes.

Definition of Done (DoD):
- `./scripts/check.sh` passes.
- `tests/contract/` and `specs/bdd/` are green with no xfails for implemented commands.
- Parity harness reports zero diffs after canonicalization, or an explicit,
  reviewed allowlist is recorded in the RFC or ticket notes.
- `xuezh-go` can run a minimal smoke path on darwin:
  - `xuezh-go db init --json`
  - `xuezh-go report hsk-audit --level hsk1 --json` (after seeding)
- The Go binary is promoted to `xuezh` and the Python binary remains available
  as rollback for one release.

## 15) Incremental delivery plan

1) Scaffold Go CLI with JSON envelope, error types, and router.
2) Implement DB init + migrations + workspace paths; pass db.init contract.
3) Implement datasets/review/events/report commands; pass BDD for each group.
4) Implement content cache commands and artifacts.
5) Implement audio convert/tts/stt/assess/process-voice with tool adapters.
6) Build parity harness; run full suite; document any allowed tolerances.
7) Cut over `xuezh` to Go after trust gate is met.
8) Keep Python binary available as rollback for one release.

## 16) Implementation order

- Define Go module layout under `src/xuezh/` (mirroring Python components).
- Port shared helpers: envelope, errors, ids, paths, jsonio, clock.
- Port DB and migrations; validate schema parity.
- Port datasets, events, reviews, reports.
- Port content cache.
- Port audio pipelines and tool adapters.
- Build and run parity harness; fix diffs.
- Swap CLI entrypoint to Go; keep Python as fallback for one release.

Rollback plan:
- If any command regresses in production usage, revert the entrypoint to Python
  and ship a hotfix with the Go binary disabled until parity is restored.

## 17) Brutal self-review (required)

- Junior engineer: Now has more concrete parity rules; still might want explicit
  references to which packages map 1:1 from Python to Go.
- Mid-level engineer: Parity tolerances are explicit; still needs a clear
  location for the parity harness code and how to run it.
- Senior/principal engineer: Audio backend risk mitigation is clearer, but
  performance tradeoffs and tool version pinning could be called out explicitly.
- PM: User-facing impact is still minimal by design; timeline realism remains
  3–5 weeks despite optimistic estimates.
- EM: Rollback plan added; staffing/milestones are still implicit.
- External stakeholder: Summary is clear; interface stability is explicit.
- End user: No behavior changes; stated more explicitly via trust gate + rollback.
