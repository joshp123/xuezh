# xuezh (Chinese Learning Engine, ZFC / Unix-style)

This repo is a **local learning engine** for Mandarin study. It is designed to be used as a **tool/skill** behind an existing Telegram bot + SOTA LLM.

## Key idea

- **Model = smart endpoint** (lesson planning, choosing what to teach next, pedagogy)
- **Engine = dumb pipes** (SQLite persistence, mechanical transforms, bounded reports, audio file materialization)

The engine must remain **ZFC-compliant**: no local ranking/selection heuristics; no “what should we do next” logic. The engine only returns primary sources and performs mechanical transforms. See `docs/reference/zfc-zero-framework-cognition.md`.

## What’s included

- `schemas/` : JSON Schemas (contract stubs; to be enforced by tickets)
- `tests/fixtures/` : minimal dataset fixtures


- `src/xuezh/` : Python package + CLI skeleton (`xuezh`)
- `tickets/` : project tickets (convert these into Beads tickets, preserving dependency + order)
- `specs/` : user requirements, BDD scenarios, and testing pyramid strategy
- `skills/chinese-learning-orchestrator/` : the Skill prompt glue (SKILL.md + references)
- `devenv.nix` : dev environment skeleton (use this; do **not** install via global package managers)
- `docs/handoff/` : handoff prompt for the implementing agent

## Quick start (developer)

1) Enter the dev environment:
   ```bash
   devenv shell
   ```

2) Install the package in editable mode:
   ```bash
   python -m pip install -e .[dev]
   ```

3) Run the CLI:
   ```bash
   xuezh --help
   xuezh version --json
   ```

4) Run tests:
   ```bash
   pytest
   ```

## Project boundaries (important)

- This repo **does not** implement the Telegram bot.
- This repo **does not** implement the lesson planner or “what to learn next” logic.
- The skill (`skills/.../SKILL.md`) teaches the model *how to use the engine*, and encodes learning best practices.

## Workspace / data path

The engine stores data under:

- `~/.clawdis/workspace/xuezh/`

Override via environment variables:
- `XUEZH_WORKSPACE_DIR`
- `XUEZH_DB_PATH`

## Ticket execution method

Implement tickets in the planned order (see `tickets/plan.yaml`) using the **RGR pattern**:
- **Red:** write/enable tests
- **Green:** minimal implementation to pass tests
- **Refactor:** clean up without behavior change

See `agents.md`.

---

## Authoritative CLI spec
See `docs/cli-contract.md`.

## Out of scope (v1)
See `specs/out-of-scope.md`.

## Authoritative specs
- CLI: `docs/cli-contract.md`
- IDs: `specs/id-scheme.md`
- Events: `specs/events.md`
- Artifact retention: `specs/artifacts/retention.md`

## CI-style checks
Run:
```bash
./scripts/check.sh
```

## Contract coverage
The repo enforces that every CLI command has a schema, a BDD scenario, and a ticket mapping.
See `tests/contract/`.
