# Test strategy (testing pyramid)

## Principle
- Most logic should be testable without the Telegram bot.
- The engine exposes a deterministic CLI. Integration tests treat the CLI as the public API.

## Pyramid layers

### Unit tests (fast, many)
Scope:
- JSON envelope shape
- deterministic ordering helpers
- schema validation
- path safety (no traversal)
- schedule transforms (if implemented as pure functions)

### Integration tests (CLI)
Scope:
- invoking `xuezh` commands
- validating exit codes
- validating JSON envelope schemas
- bounded output behavior (`--max-bytes`, `--limit`)

### E2E tests (flows)
Scope:
- temp workspace + SQLite
- import dataset -> snapshot -> review start -> grade -> report

## Mapping to BDD

- specs/bdd/review.feature:
  - unit: schedule transform helpers
  - integration: `review start`, `review grade`
  - e2e: create due item, review it, confirm due_at updated

- specs/bdd/hsk.feature:
  - integration: `report hsk`
  - e2e: import HSK dataset fixture, log exposures, report coverage

- specs/bdd/audio.feature:
  - integration: `audio convert`, `audio tts`, `audio stt`, `audio process-voice`
  - e2e: run on a fixture audio file (or mocked tool calls)

- specs/bdd/zfc.feature:
  - unit: schema lint preventing forbidden fields
  - integration: assert forbidden keys absent in outputs

## Contract sync
- `tests/contract/test_contract_coverage.py` and `tests/contract/test_contract_bdd_sync.py` ensure BDD commands, schemas, and the machine-readable contract stay aligned.

## Strict BDD mode
Set `XUEZH_STRICT_BDD=1` to make BDD scenarios fail instead of xfail when commands are unimplemented.

## No skipped tests
Skipped tests are forbidden and enforced by `tests/conftest.py`. Use xfail if truly necessary.

- specs/bdd/events.feature:
  - integration: `event log`, `event list`

- specs/bdd/gc.feature:
  - unit/e2e: retention selection + gc apply/dry-run

## Contract tests
- `tests/contract/test_contract_coverage.py`: ensures every CLI command has schema + BDD + ticket mapping.
- `tests/contract/test_contract_bdd_sync.py`: ensures BDD scenarios don't call undeclared commands.
