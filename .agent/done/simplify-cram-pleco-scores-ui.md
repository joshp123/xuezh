# Simplify Cram Review and Seed Priority From Pleco Scores

This ExecPlan is a living document. The sections `Progress`, `Surprises & Discoveries`, `Decision Log`, and `Outcomes & Retrospective` must be kept up to date as work proceeds.

This repository has `.agent/PLANS.md`; maintain this document according to that file.

## Purpose / Big Picture

After this change, Josh can open the local cram app, see the full HelloChinese and Travel Survival deck, choose a weak or unlearned batch, and review it with a stable keyboard-first UI. The local deck content comes only from the shared canonical source files that were used to make the Pleco import. Pleco itself contributes only score and recency metadata, so already-strong cards do not crowd out failed, fragile, or unseen material.

The simpler design is: import full canonical content into `cram_items`, store review state plus Pleco score metadata in `cram_state`, expose one Pleco score seed command, and fix the existing three-screen web UI. Do not add a separate score overlay table, fuzzy matching engine, sync protocol, compatibility migration chain, router, global frontend store, or HelloChinese forwarding package.

## Progress

- [x] (2026-05-02T10:57Z) Created a SQLite backup of the current local web app DB at `/Users/josh/.local/share/xuezh/backups/cram-local-20260502T105708Z.sqlite3`.
- [x] (2026-05-02T11:05Z) Inspected the latest Pleco backup and confirmed it is SQLite with `pleco_flash_cards`, `pleco_flash_categories`, `pleco_flash_categoryassigns`, `pleco_flash_scorefiles`, and `pleco_flash_scores_1`.
- [x] (2026-05-02T11:05Z) Sampled Pleco score rows and confirmed useful score metadata exists: `score`, `difficulty`, `history`, `correct`, `incorrect`, `reviewed`, `firstreviewedtime`, and `lastreviewedtime`.
- [x] (2026-05-02T11:05Z) Confirmed the current local app DB is incomplete: it has only 200 `cram_items`.
- [x] (2026-05-02T12:20Z) Confirmed `/Users/josh/Library/Mobile Documents/com~apple~CloudDocs/Full Pleco Import.txt` is the 689-card HelloChinese canonical source in Pleco import format.
- [x] (2026-05-02T12:20Z) Confirmed `/Users/josh/Library/Mobile Documents/com~apple~CloudDocs/Travel Survival Pleco Import.txt` is the 461-card Travel Survival canonical source in the same format.
- [x] (2026-05-02T12:25Z) Proved the canonical files line up with the Pleco backup when Pleco card text is used transiently: 1,075 rows match by exact source + word + sentence, and the remaining 75 rows match by source + word because Pleco merged or reused cards.
- [x] (2026-05-02T12:40Z) Rechecked the stricter constraint that Pleco must supply only score/recency metadata. A metadata-only category-assignment mapping safely seeds 1,122 rows and leaves 28 rows unseeded instead of reading Pleco card text.
- [x] (2026-05-02T12:40Z) Replaced the overbuilt score-overlay plan with this simpler score-seeding plan.
- [x] (2026-05-02T13:25Z) Replaced the HelloChinese JSONL import path with the canonical Pleco text import path.
- [x] (2026-05-02T13:25Z) Collapsed the pre-release cram schema to `cram_items` plus `cram_state`; deleted `migrations/0006_add_generic_cram.sql`.
- [x] (2026-05-02T13:25Z) Removed `internal/xuezh/hellochinese/hellochinese.go` and made CLI handlers call `internal/xuezh/cram` directly.
- [x] (2026-05-02T13:25Z) Added `xuezh pleco score-import` as a metadata-only seed command that updates `cram_state` from category assignments and `pleco_flash_scores_1`.
- [x] (2026-05-02T13:35Z) Rebuilt the local app DB from `Full Pleco Import.txt` and `Travel Survival Pleco Import.txt`, then seeded scores from the Pleco backup.
- [x] (2026-05-02T13:25Z) Finished the three-screen UI and review keyboard flow: no `/api/cram/next`, no Space shortcut, cap includes All, due counts are visible.
- [x] (2026-05-02T14:05Z) Ran `devenv shell -- ./scripts/check.sh`, built the web app through `scripts/run-cram-local.sh`, started the local app in tmux session `xuezh-cram`, and verified `GET /api/cram/overview`.

## Surprises & Discoveries

- Observation: The attached `Full Pleco Import.txt` is the better HelloChinese canonical source for this work, not the older JSONL path.
  Evidence: it has 737 lines: 689 card rows plus 48 `// HelloChinese/...` category headers, and its rows are already in the same tab-delimited Pleco import format as the Travel source.

- Observation: The current workspace does not expose a fuller checked-in JSON source for the combined deck.
  Evidence: searching the repo and the known snapshot directory found the older `hellochinese_words.jsonl` path, the current `Full Pleco Import.txt`, and the current Travel text file. The two Pleco-format text files are the available full canonical artifacts for this plan.

- Observation: The attached Travel file is also canonical and complete for this app slice.
  Evidence: it has 474 lines: 461 card rows plus 13 `// Travel Survival/...` category headers.

- Observation: Pleco and the canonical files do match cleanly if Pleco card text is read as a transient join key, but that violates the stricter product constraint.
  Evidence: comparing the Pleco backup against both text files by card text produced 1,075 exact matches by source + normalized word + normalized sentence and 75 source + normalized word matches. The stricter implementation must not use that content-matching path.

- Observation: A metadata-only mapping can still seed almost the whole deck.
  Evidence: using only Pleco root categories, child category sort order, category assignment order, card IDs, and score rows maps 1,122 of 1,150 canonical rows. `Taste` and `Going Abroad` have fewer Pleco assignments than local canonical rows, so their 28 rows should remain unseeded rather than guessed.

- Observation: Levenshtein distance should not be part of automatic matching.
  Evidence: the metadata-only mapping has no fuzzy text problem to solve. In the two mismatched categories, fuzzy matching would require reading Pleco card content and could attach scores to the wrong repeated headword.

- Observation: The actual Pleco score distribution has meaningful clusters.
  Evidence: after applying score seeds through the implemented metadata-only mapping, the buckets are `strong=82`, `seen_correct=202`, `fragile=361`, `failed=144`, and `unseen=361`. Most Travel rows are unseen, while most weak HelloChinese rows are fragile or failed.

- Observation: Pleco timestamps are ordinary Unix epoch seconds in the inspected backup.
  Evidence: sample `lastreviewedtime=1777715553` converts to `2026-05-02 11:52:33`.

- Observation: Pleco scorefiles are the correct place to read historical review metadata from.
  Evidence: Pleco's manual describes scorefiles as the collection of card scores, review history, score/easiness, review counts, and last-reviewed data used by flashcard sessions.

## Decision Log

- Decision: Use the two Pleco-format text files as canonical content for this work.
  Rationale: These are the files that line up with the Pleco backup. Importing HelloChinese from JSONL while matching Pleco from a different text shape adds avoidable mismatch.
  Date/Author: 2026-05-02 / Codex

- Decision: Keep the public commands `hellochinese import` and `travel import`, but make both use one internal Pleco text parser.
  Rationale: The command names are convenient for Josh. Internally, two parsers for the same tab-delimited format would be pointless complexity.
  Date/Author: 2026-05-02 / Codex

- Decision: Store Pleco score seed fields directly on `cram_state`.
  Rationale: The app only needs one current state row per card. A separate `cram_external_scores` table would add joins and concepts without solving a current problem.
  Date/Author: 2026-05-02 / Codex

- Decision: Match scores by Pleco category assignment order, not Pleco card text.
  Rationale: Josh's constraint is that Pleco supplies scoring/recency metadata only. Category roots, child category sort, assignment IDs, card IDs, and score rows are enough to seed 1,122 rows safely. The remaining 28 rows stay unseeded instead of reading Pleco definitions or guessing.
  Date/Author: 2026-05-02 / Codex

- Decision: Do not implement Levenshtein matching.
  Rationale: Fuzzy matching is unnecessary for the metadata-only plan and would require using Pleco card content as an identity source.
  Date/Author: 2026-05-02 / Codex

- Decision: Use distribution-based score buckets, not a single `score <= 600` cutoff.
  Rationale: The observed backup has clear score/history clusters. The UI should prioritize failed and fragile cards before merely seen-correct or strong cards.
  Date/Author: 2026-05-02 / Codex

- Decision: Treat `pleco score-import` as a one-way seed, not a sync feature.
  Rationale: This work needs initial prioritization from Pleco. Two-way sync or repeated merge semantics after local xuezh reviews would be extra product surface.
  Date/Author: 2026-05-02 / Codex

## Outcomes & Retrospective

Implementation is complete for this slice. The code imports the two canonical Pleco-format text sources into one `cram_items` schema, seeds Pleco score metadata directly into `cram_state` without reading Pleco card content, and keeps the web UI on the simpler overview/batch/grade API surface. The local app DB was rebuilt with 1,150 canonical cards, 1,122 seeded rows, 28 unseeded rows, and bucket counts `strong=82`, `seen_correct=202`, `fragile=361`, `failed=144`, `unseen=361`. The app is running at `http://127.0.0.1:8765/`.

## Context and Orientation

The repo is `/Users/josh/code/xuezh`. The Go CLI entrypoint is `cmd/xuezh-go/main.go`, and the command switch is `internal/xuezh/cli/cli.go`. Database setup uses `internal/xuezh/db/db.go`, `internal/xuezh/migrations/migrations.go`, and SQL files under `migrations/`. The cram engine work lives in `internal/xuezh/cram/`. The web frontend is in `web/src/main.tsx` and `web/src/styles.css`. The local runner is `scripts/run-cram-local.sh`.

The canonical content sources are the shared generated source files currently available in this workspace:

    /Users/josh/Library/Mobile Documents/com~apple~CloudDocs/Full Pleco Import.txt
    /Users/josh/Library/Mobile Documents/com~apple~CloudDocs/Travel Survival Pleco Import.txt

`Full Pleco Import.txt` contains HelloChinese cards. `Travel Survival Pleco Import.txt` contains Travel Survival cards. These are not read from the Pleco backup. They are the canonical local content sources used to build `cram_items`. A card row is tab-delimited: Chinese headword, pronunciation, then a Pleco-format payload. The payload uses the private separator U+EAB1 and currently contains English meaning, a blank field, Chinese sentence, English sentence, and sometimes a note.

The latest Pleco backup inspected for this plan is:

    /Users/josh/Downloads/Pleco Flash Backup 260502.pqb

`Canonical card` means one row in `cram_items`, displayed in xuezh. `Score seed` means Pleco review metadata copied into `cram_state` before Josh starts reviewing in xuezh. `Score bucket` means a simple priority label derived from Pleco score facts; it is not a reimplementation of Pleco's scheduler. `Metadata-only Pleco import` means the implementation may read Pleco score rows, category rows, category assignment rows, and card IDs, but must not read or persist Pleco headwords, pronunciations, definitions, sentences, or English meanings.

## Plan of Work

First, simplify the schema. Do not add a new migration. This repo is pre-release, and the local DB has already been backed up. Edit the current cram schema directly so a fresh database has `cram_items` and `cram_state`. Delete `migrations/0006_add_generic_cram.sql`, and replace `migrations/0005_add_hellochinese_cram.sql` with the current cram schema.

`cram_items` should hold only canonical display content:

    id TEXT PRIMARY KEY
    source TEXT NOT NULL
    category TEXT NOT NULL
    learning_order INTEGER NOT NULL
    source_index INTEGER
    pinyin TEXT NOT NULL
    hanzi TEXT NOT NULL
    meaning TEXT NOT NULL
    sentence_pinyin TEXT
    sentence_hanzi TEXT NOT NULL
    sentence_hanzi_raw TEXT
    sentence_meaning TEXT
    row_hash TEXT NOT NULL UNIQUE
    sentence_audio_paths_json TEXT NOT NULL DEFAULT '{}'
    created_at TEXT NOT NULL
    updated_at TEXT NOT NULL
    UNIQUE(source, learning_order)
    UNIQUE(source, hanzi, sentence_hanzi)

`cram_state` should hold xuezh review state plus the Pleco seed fields:

    item_id TEXT PRIMARY KEY
    status TEXT NOT NULL
    next_due_at TEXT
    interval_minutes INTEGER NOT NULL DEFAULT 0
    seen_count INTEGER NOT NULL DEFAULT 0
    lapse_count INTEGER NOT NULL DEFAULT 0
    last_grade TEXT
    pleco_card_id INTEGER
    pleco_score INTEGER
    pleco_difficulty INTEGER
    pleco_history TEXT
    pleco_correct_count INTEGER
    pleco_incorrect_count INTEGER
    pleco_reviewed_count INTEGER
    pleco_first_reviewed_at TEXT
    pleco_last_reviewed_at TEXT
    score_bucket TEXT NOT NULL DEFAULT 'unseen'
    score_imported_at TEXT
    created_at TEXT NOT NULL
    updated_at TEXT NOT NULL

Second, collapse the content import path in `internal/xuezh/cram/import.go`. Keep `ImportHelloChinese` and `ImportTravelSurvival`, but have both call one internal parser for the shared Pleco-format text rows. That parser should strip a required root prefix from categories: `HelloChinese/Hello` becomes `Hello`, and `Travel Survival/Must Know - Restaurant Vendor Flow` becomes `Must Know - Restaurant Vendor Flow`. It should normalize artificial spaces and ASCII `?` / `!` in Chinese sentences the same way for both sources. Remove the HelloChinese JSONL parser from this local cram path unless another test proves it is still required. Update `internal/xuezh/cram/cram_test.go` because it currently expects the Travel category to keep the `Travel Survival/` prefix.

Third, remove the forwarding package. Delete `internal/xuezh/hellochinese/hellochinese.go`. In `internal/xuezh/cli/cli.go`, call `cram.ImportHelloChinese`, `cram.ImportTravelSurvival`, `cram.BackfillAudio`, `cram.NextCards`, and `cram.GradeCard` directly. Move any useful tests from `internal/xuezh/hellochinese/` into `internal/xuezh/cram/`, using Pleco text fixtures instead of JSONL fixtures.

Fourth, add `internal/xuezh/cram/pleco.go` for score seeding. The command opens the `.pqb` backup read-only. It reads `pleco_flash_scorefiles` to confirm the default scorefile, `pleco_flash_scores_1` for score and recency facts, `pleco_flash_categories` for the source/category tree, and `pleco_flash_categoryassigns` for the sequence of Pleco card IDs inside each category. It must not read `pleco_flash_cards.hw`, `pleco_flash_cards.pron`, or `pleco_flash_cards.defn`, because those are card content.

The metadata-only matching rule is deliberately small:

1. Find the Pleco root category named `HelloChinese` and map it to local source `hellochinese`; find root category `Travel Survival` and map it to local source `travel_survival`.
2. For each child category under those roots, map by category name and sort order to local `cram_items.source + cram_items.category`.
3. For a category whose Pleco assignment count equals the local row count, order local rows by `learning_order`, order Pleco assignments by assignment `id`, and copy the score row for each assignment's `card` ID to the corresponding local row.
4. For a category whose counts differ, do not guess. Leave every row in that category with `score_bucket = unseen`, and report the category in `unseeded_categories`.

The expected result on the inspected files is 1,122 seeded rows and 28 unseeded rows. The unseeded rows are all in HelloChinese categories `Taste` and `Going Abroad`, where Pleco has fewer category assignments than the canonical local file. This is acceptable because weak or unknown unseeded rows still appear as due; the important part is not attaching a score to the wrong row. Do not add Levenshtein matching and do not read Pleco card text to recover these 28 rows.

Fifth, define `scoreBucketFromPleco` in `internal/xuezh/cram/pleco.go`. This is not Pleco's scheduler. It is only an import-time priority label:

    if there is no score row or reviewed count is zero:
      bucket = unseen

    else if score <= 100:
      bucket = failed

    else if incorrect count > 0:
      bucket = fragile

    else if score == 600 and incorrect count == 0:
      bucket = seen_correct

    else if score >= 1000 and correct count > 0:
      bucket = strong

    else:
      bucket = fragile

Seed `cram_state` from the bucket. `strong` cards get `status = review`, `interval_minutes = 10080`, and `next_due_at = pleco_last_reviewed_at + 7 days`. All other buckets get `status = learning` except `unseen`, which gets `status = new`; both should have `next_due_at = NULL` so they are eligible. Set `seen_count` to at least Pleco reviewed count and `lapse_count` to at least Pleco incorrect count.

Sixth, add a narrow CLI command:

    xuezh pleco score-import --path "/Users/josh/Downloads/Pleco Flash Backup 260502.pqb" --json

The JSON result should include `canonical_rows`, `seeded_rows`, `unseeded_rows`, `unseeded_categories`, and `bucket_counts`. It should fail clearly if local `review_events` already contains `event_type = 'cram.grade'`, because this command is a seed step and not a sync step.

Seventh, update overview and batch selection. `internal/xuezh/cram/query.go` should keep `OverviewFor(now)` and `NextCards(opts, now)`, but order eligible cards by bucket priority before learning order:

    failed
    fragile
    unseen
    seen_correct
    strong only when actually due by next_due_at

`OverviewFor` should return only what the UI needs: `total_count`, `learned_count`, and `eligible_count` for each source and category. Before local xuezh review starts, `learned_count` comes from `score_bucket = strong`. After a card has a local `last_grade`, local state wins: count it learned only when `interval_minutes >= 1440`. `eligible_count` is the count that `NextCards` could return now.

Eighth, keep the two-button scheduler simple in `internal/xuezh/cram/scheduler.go`. The backend accepts only `hard` and `easy`. `hard` records a lapse and makes the item due soon. `easy` advances the interval. The UI can keep a retry pile inside the current browser session so hard cards return later in the same batch, but React should not contain interval policy.

Ninth, simplify and finish the UI in `web/src/main.tsx` and `web/src/styles.css`. Keep no router, no global store, no component framework, and no extra screens. In `internal/xuezh/webserver/webserver.go`, remove `GET /api/cram/next` and `handleNext`; the web UI should use only `GET /api/cram/overview`, `POST /api/cram/batch`, and `POST /api/cram/grade`.

The UI remains three screens:

Home shows only source-level status and one `Choose batch` action, for example `HelloChinese 80/689 learned · 609 due` and `Travel Survival 2/461 learned · 459 due` after the inspected Pleco seed.

Batch shows category rows with checkboxes, source label, `X/Y learned`, `N due`, a live selected-card count, cap controls for `50`, `100`, `200`, `500`, and `All`, and a Start button. The selected count must use backend eligible counts, not total category size.

Review keeps the Chinese word and sentence fixed before and after reveal. Reserve answer height so the sentence and buttons do not move. Use `Enter` or `0` to reveal/continue, `5` to replay, `1` for Hard, and `2` for Easy. Numpad keys and regular number keys should both work. Remove the current Space-key reveal behavior in `web/src/main.tsx`; do not make Space a shortcut.

## Concrete Steps

Run all commands from `/Users/josh/code/xuezh`.

Verify the backup exists:

    ls -lh /Users/josh/.local/share/xuezh/backups/cram-local-20260502T105708Z.sqlite3

Inspect the current worktree before editing:

    git status --short
    git --no-pager diff --stat

Update these backend files:

    migrations/0005_add_hellochinese_cram.sql
    internal/xuezh/cram/types.go
    internal/xuezh/cram/db.go
    internal/xuezh/cram/import.go
    internal/xuezh/cram/query.go
    internal/xuezh/cram/scheduler.go
    internal/xuezh/cram/pleco.go
    internal/xuezh/cli/cli.go
    internal/xuezh/webserver/webserver.go

Delete or stop using:

    migrations/0006_add_generic_cram.sql
    internal/xuezh/hellochinese/hellochinese.go

Update public contract artifacts for changed commands:

    docs/cli-contract.md
    specs/cli/contract.json
    specs/implemented-commands.json
    schemas/hellochinese.import.schema.json
    schemas/travel.import.schema.json
    schemas/pleco.score-import.schema.json
    schemas/cram.next.schema.json
    schemas/cram.grade.schema.json
    schemas/cram.overview.schema.json
    specs/bdd/hellochinese.feature

Update local runner defaults:

    scripts/run-cram-local.sh

It should import:

    /Users/josh/Library/Mobile Documents/com~apple~CloudDocs/Full Pleco Import.txt
    /Users/josh/Library/Mobile Documents/com~apple~CloudDocs/Travel Survival Pleco Import.txt

Run formatting and tests:

    gofmt -w internal/xuezh/cram internal/xuezh/cli/cli.go internal/xuezh/webserver/webserver.go
    devenv shell -- ./scripts/check.sh
    devenv shell -- sh -lc 'cd web && pnpm build'

Rebuild the local cram DB only after tests pass and only after the backup file exists. If `trash` is available, move the old DB aside:

    trash "$HOME/.local/share/xuezh/cram-local/db.sqlite3"

If `trash` is unavailable, move it to the backup directory with a timestamp instead of deleting it.

Then import the local deck:

    XUEZH_WORKSPACE_DIR="$HOME/.local/share/xuezh/cram-local" \
      devenv shell -- go run ./cmd/xuezh-go hellochinese import \
      --path "/Users/josh/Library/Mobile Documents/com~apple~CloudDocs/Full Pleco Import.txt" \
      --audio none --json

    XUEZH_WORKSPACE_DIR="$HOME/.local/share/xuezh/cram-local" \
      devenv shell -- go run ./cmd/xuezh-go travel import \
      --path "/Users/josh/Library/Mobile Documents/com~apple~CloudDocs/Travel Survival Pleco Import.txt" \
      --audio none --json

    XUEZH_WORKSPACE_DIR="$HOME/.local/share/xuezh/cram-local" \
      devenv shell -- go run ./cmd/xuezh-go pleco score-import \
      --path "/Users/josh/Downloads/Pleco Flash Backup 260502.pqb" --json

Start the app:

    ./scripts/run-cram-local.sh

Open:

    http://127.0.0.1:8765/

## Validation and Acceptance

The backend is acceptable when `devenv shell -- ./scripts/check.sh` passes and `devenv shell -- sh -lc 'cd web && pnpm build'` passes.

The content imports are acceptable when a fresh DB has exactly 689 `hellochinese` rows and 461 `travel_survival` rows in `cram_items`, for 1,150 total canonical cards.

The Pleco score seed is acceptable when `pleco score-import` reports:

    canonical_rows: 1150
    seeded_rows: 1122
    unseeded_rows: 28
    unseeded_categories:
      - hellochinese / Taste
      - hellochinese / Going Abroad
    bucket_counts:
      strong: 82
      seen_correct: 202
      fragile: 361
      failed: 144
      unseen: 361

Small count differences are acceptable only if the input files or Pleco backup path changed; otherwise treat differences as a bug.

The matching is acceptable when the importer can seed the first HelloChinese category by category order without reading card content: local `Hello` row 1 receives the score row for the first Pleco assignment under the current `HelloChinese/Hello` category, local `Hello` row 2 receives the second, and so on. It is also acceptable when the importer refuses to seed `Taste` and `Going Abroad` because their local row counts do not match Pleco assignment counts.

The UI is acceptable when Home shows source-level learned/due counts, Batch selected counts update live from eligible counts, Review reveal does not move the Chinese sentence, audio replay works when audio exists, keyboard-only review works with `Enter`/`0`, `5`, `1`, and `2`, and pressing Space does nothing.

## Idempotence and Recovery

Canonical imports must be safe to rerun. Reimporting the same text files should report existing rows, not duplicate rows.

`pleco score-import` is safe before any xuezh review events exist. After local `cram.grade` events exist, it should fail clearly instead of merging old Pleco state over newer xuezh state. This keeps the first implementation honest and avoids a fake sync feature.

The local app DB has already been backed up at:

    /Users/josh/.local/share/xuezh/backups/cram-local-20260502T105708Z.sqlite3

If the rebuild goes wrong, stop the app, move the bad DB aside, and restore with:

    sqlite3 "$HOME/.local/share/xuezh/cram-local/db.sqlite3" \
      ".restore '/Users/josh/.local/share/xuezh/backups/cram-local-20260502T105708Z.sqlite3'"

Do not use `git reset --hard` or destructive Git commands.

## Artifacts and Notes

Canonical source counts:

    Full Pleco Import.txt: 737 lines = 689 cards + 48 category headers
    Travel Survival Pleco Import.txt: 474 lines = 461 cards + 13 category headers

Pleco backup counts:

    pleco_flash_cards: 1074
    pleco_flash_scores_1: 741 scored card IDs
    HelloChinese source memberships: 663
    Travel Survival source memberships: 461
    card IDs under both source roots: 50

Observed metadata-only score seeding against canonical rows:

    category assignment rows mapped safely: 1122
    unseeded canonical rows: 28
    unseeded categories: hellochinese / Taste, hellochinese / Going Abroad

Observed text-based matching, used only as evidence and not as the implementation rule:

    exact source + word + sentence: 1075
    word-shared source + word: 75
    unmatched canonical words: 0

Observed score distribution after applying metadata-only score seeds to canonical rows:

    strong: 82
    seen_correct: 202
    fragile: 361
    failed: 144
    unseen: 361

Observed raw score clusters:

    score 600 / difficulty 94: 309 Pleco score rows
    score 600 / difficulty 104: 187 rows
    score 100 / difficulty 90: 134 rows
    score around 21800 / difficulty 112: 79 rows

Observed common history strings:

    62: 126 rows
    622: 124 rows
    6: 118 rows
    222: 71 rows
    66: 69 rows
    662: 50 rows

In this backup, history digit `6` represents a correct answer quality and `2` represents an incorrect answer quality in the simple scoring setup. The bucket logic uses this only as a priority clue.

## Interfaces and Dependencies

In `internal/xuezh/cram/types.go`, add:

    type PlecoScoreImportOptions struct {
        Path string
    }

    type PlecoScoreImportResult struct {
        CanonicalRows      int            `json:"canonical_rows"`
        SeededRows         int            `json:"seeded_rows"`
        UnseededRows       int            `json:"unseeded_rows"`
        UnseededCategories []string       `json:"unseeded_categories"`
        BucketCounts       map[string]int `json:"bucket_counts"`
    }

In `internal/xuezh/cram/pleco.go`, define:

    func ImportPlecoScores(opts PlecoScoreImportOptions) (PlecoScoreImportResult, error)

This function hides every Pleco-specific detail from CLI and web callers: opening the backup, mapping category assignment card IDs to canonical rows by order, bucketing score facts, and seeding `cram_state`.

In `internal/xuezh/cram/query.go`, keep:

    func OverviewFor(now time.Time) (Overview, error)
    func NextCards(opts NextOptions, now time.Time) ([]Card, error)

These functions hide SQL and bucket ordering from the web server and CLI.

In `internal/xuezh/cram/scheduler.go`, keep:

    func GradeCard(opts GradeOptions, now time.Time) (GradeResult, error)

This function hides review interval updates and event logging from the web UI.

In `internal/xuezh/webserver/webserver.go`, keep the API small:

    GET  /api/cram/overview
    POST /api/cram/batch
    POST /api/cram/grade

No Pleco web endpoint is needed. Pleco score import is a CLI/admin seed step.

## Change Note

Updated on 2026-05-02 after code and data review of the user's comments. The plan now uses the attached canonical Pleco text files instead of the older HelloChinese JSONL path, removes the extra score overlay table, removes automatic fuzzy matching, replaces the crude `score <= 600` rule with buckets from the observed Pleco score distribution, and treats `pleco score-import` as a one-way seed instead of a sync system.

Updated again on 2026-05-02 to enforce the stricter constraint that Pleco contributes scoring and recency metadata only. The score import no longer reads Pleco card text for matching; it maps scores by category assignment order, seeds 1,122 rows, leaves the 28 count-mismatched rows unseen, and explicitly removes the extra web `next` endpoint plus Space-key reveal behavior.
