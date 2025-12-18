# Spec invariants (must remain true)

These are non-negotiable “idiot-proof” constraints. If you violate one, you must update:
- the CLI contract
- schemas
- BDD specs
- tests
and explicitly justify the change in a ticket.

## Invariants
- ID scheme is authoritative (`specs/id-scheme.md`).
- Single-user system (workspace == one learner). No `--user`.
- ZFC boundary: no recommendation fields, no heuristic ranking/scoring.
- All commands return JSON envelope (`ok` or `err`).
- Bounded outputs (`--limit` / `--max-bytes`) with spill-to-artifact behavior.
- CLI contract is authoritative (`docs/cli-contract.md`, `specs/cli/contract.json`).
- BDD specs are executable and in sync with CLI contract and schemas.
- Events are recorded as facts (`specs/events.md`).
- Artifact retention policy is enforced (`specs/artifacts/retention.md`).
- No skipped tests (enforced by `tests/conftest.py`).
