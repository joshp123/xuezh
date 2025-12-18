# Typed errors + exit codes (policy)

The `xuezh` CLI is a tool surface used by an LLM Skill. It must be safe to automate.

## Typed errors

All failures produced by the engine are returned as an **ERR envelope**:

- `ok: false`
- `error.type`: **stable string** used for programmatic branching
- `error.message`: human-readable summary
- `error.details`: structured details (bounded)

### Adding a new error type

This repo intentionally avoids a large upfront taxonomy. Error types are added **only when needed**.

When introducing a new `error.type`:
1) Add it to `src/xuezh/core/error_types.py` (`KNOWN_ERROR_TYPES`).
2) Use the same string consistently across commands.

## Exit codes

The CLI uses exit codes for shell/operator ergonomics:
- `ok: true` → exit code **0**
- `ok: false` → exit code **non-zero**

The JSON envelope remains the primary API; exit codes are a secondary signal.
