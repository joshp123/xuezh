# Documentation map (what is authoritative)

This repo is intentionally **contract-first**. Not all documents have the same “authority”.

## Authoritative (must stay in sync)

If you change any one of these, you usually must update the others:

1) **CLI contract (human)**: `docs/cli-contract.md`  
2) **CLI contract (machine)**: `specs/cli/contract.json`  
3) **Output schemas**: `schemas/`  
4) **Executable BDD specs**: `specs/bdd/`  
5) **Contract sync tests**: `tests/contract/`

Tickets in `tickets/` are the implementation plan, but the CLI surface is defined by the contract.

## High-signal supporting specs

These are strong guidance and should generally be treated as binding:
- ZFC boundary + invariants: `specs/invariants.md`, `docs/architecture.md`
- North stars + URs: `specs/north-stars.md`, `specs/user-requirements.md`
- IDs, events, retention: `specs/id-scheme.md`, `specs/events.md`, `specs/artifacts/retention.md`

## Background / reference (NOT a spec)

`docs/reference/*` contains background material and prior thinking. It is **not** a requirements source of truth.
