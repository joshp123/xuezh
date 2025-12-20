# HSK scope decision (v1)

**Goal:** keep the HSK audit useful while avoiding “character list” fetishism.

## Default (v1)
- HSK audit focuses on:
  - **vocabulary (words)**
  - **grammar points**
- Characters are **out of scope** for v1.

Rationale:
- Practical Mandarin proficiency is mostly word/chunk retrieval.
- Character-level auditing can be added later without affecting the core loop.

## Hold point (requires user sign-off)
Before implementing `report.hsk`, the agent must:
1) show a sample output JSON structure using **real datasets**
2) confirm with the user whether to:
   - include chars or not
   - how to handle upstream HSK 7-9 levels

## Approved decisions (2025-12-18)
## Approved decisions (2025-12-19)
- HSK audit is **vocab + grammar only** (no chars in v1).
- Default seed imports **levels 1–6**; upstream `7-9` rows are excluded unless explicitly imported.
- `report.hsk --level 7-9` targets the bucket when present.
- Coverage and counts include **known/unknown** splits per level.
