# HSK scope decision (v1)

**Goal:** keep the HSK audit useful while avoiding “character list” fetishism.

## Default (v1)
- HSK audit focuses on:
  - **vocabulary (words)**
  - **grammar points**
- Characters are **optional**:
  - can be included if `--include-chars` and `hsk_chars` dataset exists
  - otherwise omitted

Rationale:
- Practical Mandarin proficiency is mostly word/chunk retrieval.
- Character-level auditing can be added later without affecting the core loop.

## Hold point (requires user sign-off)
Before implementing `report.hsk`, the agent must:
1) show a sample output JSON structure using fixture datasets
2) confirm with the user whether to:
   - keep chars optional
   - drop chars entirely
   - include chars by default
