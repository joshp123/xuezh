# Tests

This repo follows a testing pyramid:

- unit: pure functions / deterministic helpers
- integration: CLI invocation & JSON contract
- e2e: full flows with temp workspace and SQLite

See `specs/test-strategy.md` for the mapping to BDD scenarios.
