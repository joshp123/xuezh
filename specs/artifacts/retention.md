# Artifact retention policy (authoritative)

The engine creates artifacts (audio, JSON spills, exports) under the workspace.

This policy is **mechanical** (ZFC-safe): it is budget enforcement, not “intelligence”.

## Workspace layout

- `artifacts/` — tool outputs and spilled large outputs
- `cache/` — cached generated content
- `exports/` — user-requested exports (weekly)
- `backups/` — DB backups

Artifacts under `artifacts/` should be stored in date buckets:
- `artifacts/YYYY/MM/DD/<artifact_id>.<ext>`

## Default retention windows

Defaults:
- `artifacts/`: keep 90 days
- `backups/`: keep 30 days
- `exports/`: keep 180 days
- `cache/`: keep 180 days (or size-capped, but deterministic)

Retention is not runtime-configurable today. Add a config-file key only when a
real retention tuning need exists.

## Garbage collection command

The engine must provide a deterministic GC command:

- `xuezh gc --dry-run --json`
- `xuezh gc --apply --json`

Behavior:
- Never delete outside workspace.
- Deterministically select files older than retention window.
- Return a bounded list of files deleted (or would delete), and totals.

## Testing requirements

- Unit tests: path safety, selection logic (given fake timestamps)
- E2E tests: create temp files with fake mtimes and verify GC deletes only intended ones
