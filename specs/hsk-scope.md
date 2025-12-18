# HSK scope decision (v1)

**Goal:** keep the HSK audit useful while avoiding “character list” fetishism.

## Default (v1)
- HSK audit focuses on:
  - **vocabulary (words)**
  - **grammar points**
- Characters are **optional**:
  - included only if `--include-chars` and `hsk_chars` dataset exists
  - otherwise omitted

Rationale:
- Practical Mandarin proficiency is mostly word/chunk retrieval.
- Character-level auditing can be added later without affecting the core loop.

## Hold point (requires user sign-off)
Before implementing `report.hsk`, the agent must:
1) show a sample output JSON structure using **real datasets**
2) confirm with the user whether to:
   - keep chars optional
   - drop chars entirely
   - include chars by default

## Approved decisions (2025-12-18)
- Chars are **optional** (only included with `--include-chars`)
- `level` may be a string to preserve upstream values like `"7-9"`
- Coverage and counts include **known/unknown** splits per level
