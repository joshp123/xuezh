# Tickets

Each ticket is a standalone markdown file with YAML frontmatter.

## Source of truth
- CLI contract: `docs/cli-contract.md` and `specs/cli/contract.json`
- Output schemas: `schemas/`
- BDD specs: `specs/bdd/*.feature` (executable via pytest-bdd)

## Execution order
These files are historical ticket specs. If they are used for context, preserve:
- dependencies (`depends_on`)
- blocks (`blocks`)
- **desired order** (`order`)

See `tickets/plan.yaml`.

## v4 additions
- T-01B: ID scheme enforcement
- T-02B: Artifact retention + GC
- T-04A: HSK hold point (requires user sign-off)
- T-06B: Event logging (exposures)
