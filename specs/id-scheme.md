# ID scheme (authoritative)

This project must be **idiot-proof**: IDs must be unambiguous, stable, and easy to validate.

## Item IDs (words / grammar / characters)

Items are referenced in the CLI and stored in SQLite using **typed IDs**.

### Format
- Words: `w_<12hex>`
- Grammar points: `g_<12hex>`
- Characters (optional): `c_<12hex>`

Where `<12hex>` is the first 12 hex characters of:

- word: `sha1("word|" + hanzi + "|" + pinyin_normalized)`
- grammar: `sha1("grammar|" + grammar_id)`
- char: `sha1("char|" + character)`

**Rationale:** deterministic IDs allow idempotent dataset imports and stable references across runs.

### Pinyin normalization
- For ID hashing, normalize pinyin to a consistent form:
  - lowercased
  - trim whitespace
  - collapse multiple spaces to one
  - keep tone marks OR convert to numbered tones, but pick one and document it in implementation.

The exact normalization function must be tested and frozen early.

## Event IDs

Events are append-only primary sources stored by the engine.

- Event IDs: `ev_<26>` using ULID (preferred) or `ev_<32hex>` UUID.
- Must be unique and sortable by time (ULID recommended).

## Content IDs

Cached generated content is referenced by:

- Content IDs: `ct_<12hex>` = first 12 hex of `sha1(type + "|" + key)`

## Artifact IDs

Artifacts are primarily referenced by their workspace-relative paths.
When an explicit ID is needed:

- Artifact ID: `ar_<12hex>` = first 12 hex of `sha1(path)`

## Validation
- All IDs must be ASCII.
- Regex:
  - word: `^w_[0-9a-f]{12}$`
  - grammar: `^g_[0-9a-f]{12}$`
  - char: `^c_[0-9a-f]{12}$`
  - event: `^ev_[0-9A-Z]{26}$` (ULID) OR `^ev_[0-9a-f]{32}$` (UUID)

These regexes must be enforced by unit tests and by schema validation where applicable.
