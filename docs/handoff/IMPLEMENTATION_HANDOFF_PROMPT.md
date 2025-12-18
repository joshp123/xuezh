# Implementation handoff prompt

You are an autonomous coding agent implementing this repo.

## Mission
Implement the `chlearn` learning engine as a **ZFC/Unix-style dumb pipe** used by a SOTA LLM + Telegram bot.

- The LLM does lesson planning and decides what to teach next.
- The engine stores primary sources, provides bounded reports, and materializes audio artifacts.

## Required workflow
1) Convert all files in `tickets/` into Beads tickets (preserve dependencies + explicit order).
   - Preserve:
     - dependencies (blocking relationships)
     - **desired implementation order** (see `tickets/plan.yaml`)
   - The Beads representation must encode both dependencies and ordering.

2) Implement tickets **in order** using **RGR (Red-Green-Refactor)**.
   - Add/enable tests for each acceptance criteria before implementation.
   - Keep tests aligned to `specs/bdd/*.feature` and the testing pyramid (`specs/test-strategy.md`).

3) Use `devenv` for dependencies.
   - Do not use brew/global installs.
   - If you need a tool, update `devenv.nix`.

4) Maintain ZFC boundary (critical):
   - Do not add heuristic ranking or “what next” logic to the engine.
   - Engine returns facts + candidate sets + mechanical schedule state only.

## Deliverables
- A working `chlearn` CLI with JSON envelope outputs.
- SQLite persistence in workspace path.
- Snapshot + report commands (bounded outputs).
- Audio pipeline wrappers for format conversion, TTS, STT, and assessment (v0 local; v1 optional paid).
- Tests at unit/integration/e2e levels per BDD scenarios.

## Reference documents
- `docs/reference/zfc-zero-framework-cognition.md`
- `docs/reference/oracle-mega-prompt.md`
- `skills/chinese-learning-orchestrator/SKILL.md`


## Additional must-follow rules
- Single-user: do not implement multi-user flags.
- CLI contract is authoritative: keep `docs/cli-contract.md`, `specs/cli/contract.json`, `schemas/`, and `specs/bdd/*.feature` in sync.


## Git bootstrap (must do before tickets)

1) `git init`
2) Commit initial scaffold
3) Create a private GitHub repo with `gh` and push upstream
4) Then proceed with ticket conversion + implementation

## Commit discipline
- One ticket == one atomic commit (no fixups).
- Run `./scripts/check.sh` before every commit.
- Zero skipped tests is enforced by the test suite.
