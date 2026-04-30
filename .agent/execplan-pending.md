# Build a CLI-First HelloChinese Cram Review Engine

This ExecPlan is a living document. The sections `Progress`, `Surprises & Discoveries`, `Decision Log`, and `Outcomes & Retrospective` must be kept up to date as work proceeds.

This repository now has `.agent/PLANS.md`; maintain this document according to that file.

## Purpose / Big Picture

After this change, Josh can import a normalized HelloChinese word list, start a fast review session from the command line, hear the canonical sentence audio automatically, review the target word in its original Chinese sentence context, grade the item, and move quickly through hundreds of initial items. The first visible success is a command that imports `/Users/josh/Documents/Codex/2026-04-24/can-you-use-the-hellochinese-plugin/snapshots/hellochinese_words.jsonl`, generates or verifies audio for the canonical sentences, then returns the first due review card with Chinese sentence, target word, pinyin, English answer, and audio artifact path.

The current repo already has SQLite, migrations, JSON envelopes, TTS, review state, and a Go CLI. The missing piece is a simpler surface for this specific use case. Today a caller would have to understand dataset imports, word IDs, generic SRS commands, and audio commands separately. This plan creates a deep `cram`/HelloChinese module that hides that sequencing: import rows, preserve learning order, prepare sentence audio, select the next item, and apply the simple cram schedule.

## Progress

- [x] (2026-04-26T13:49Z) Inspected the HelloChinese JSONL shape and confirmed it has 689 rows, complete `index` values from 1 to 689, and no duplicate `hanzi` or duplicate `hanzi + sentence_hanzi` pairs.
- [x] (2026-04-26T13:49Z) Inspected the existing Go CLI, SQLite migration flow, SRS module, ID helpers, audio TTS implementation, and test command.
- [x] (2026-04-26T13:49Z) Created this ExecPlan with concrete commands, UX decisions, schema shape, and validation steps.
- [x] (2026-04-26T13:49Z) Smoke-tested the existing TTS command in the normal shell and confirmed it fails safely because `edge-tts` is not on `PATH`.
- [x] (2026-04-26T14:15Z) Entered the repo `devenv` correctly with `devenv shell sh -lc ...`, confirmed `edge-tts` and `ffmpeg` are available there, and generated one sentence Ogg/Opus file successfully.
- [x] (2026-04-26T14:15Z) Audited the plan against actual repo details using the `execplan-improve` workflow and strengthened the audio/tooling and contract sections.
- [x] (2026-04-26T15:20Z) Add the new migration for HelloChinese corpus rows and cram review state.
- [x] (2026-04-26T15:20Z) Implement import, audio preparation, next-card selection, and grading in a new small package.
- [x] (2026-04-26T15:20Z) Add CLI commands and JSON schemas for the new behavior.
- [x] (2026-04-26T15:20Z) Add tests and run `./scripts/check.sh`.
- [x] (2026-04-26T15:20Z) Add the tiny web wrapper after the CLI behavior works end to end.

## Surprises & Discoveries

- Observation: The current full corpus file is better than the earlier partial snapshot.
  Evidence: `wc -l` showed 689 JSONL rows; checking `.index` showed min 1, max 689, present 689, missing 0.
- Observation: The repo already has `audio.TTSAudio`, which writes workspace-relative artifacts via `edge-tts` and `ffmpeg`.
  Evidence: `internal/xuezh/audio/audio.go` defines `TTSAudio(text, voice, outPath, backend, purpose string)`.
- Observation: Existing generic SRS state lives in `user_knowledge` and `review_events`, but this feature should not force the web/CLI caller to use low-level item IDs and numeric grades directly.
  Evidence: `internal/xuezh/srs/srs.go` exposes generic `ListDueItems`, `ScheduleNextDue`, and `UpsertKnowledge`, while the target UX needs `again|hard|good|easy` over imported HelloChinese rows.
- Observation: The current normal shell cannot generate TTS audio because `edge-tts` is missing, while `ffmpeg` is present.
  Evidence: `XUEZH_WORKSPACE_DIR=/tmp/xuezh-audio-smoke go run ./cmd/xuezh-go audio tts --text '你 是 龙大。' --voice XiaoxiaoNeural --out artifacts/smoke/sentence.ogg --backend edge-tts --json` returned `ok:false` with error type `TOOL_MISSING` and message `Required tool not found on PATH: edge-tts`.
- Observation: The repo `devenv` does provide the audio tools when entered correctly.
  Evidence: `devenv shell sh -lc 'command -v edge-tts; command -v ffmpeg'` printed `/nix/store/...-python3.13-edge-tts-7.2.3/bin/edge-tts` and `/nix/store/...-ffmpeg-8.0-bin/bin/ffmpeg`.
- Observation: The existing TTS flow works in the repo `devenv`, but using `/tmp` as `XUEZH_WORKSPACE_DIR` can hit a macOS path-safety issue because `/tmp` resolves to `/private/tmp`.
  Evidence: `XUEZH_WORKSPACE_DIR=/private/tmp/xuezh-audio-smoke-devenv go run ./cmd/xuezh-go audio tts --text "你是龙大。" --voice XiaoxiaoNeural --out artifacts/smoke/sentence.ogg --backend edge-tts --json` returned `ok:true` with artifact `artifacts/smoke/sentence.ogg`; `file` reported `Ogg data, Opus audio, mono, 48000 Hz`.
- Observation: There is currently no `tests/` directory or checked-in Go test file in this working tree, despite docs mentioning tests.
  Evidence: `rg --files -g '*_test.go' -g 'tests/**'` returned no files.
- Observation: There is currently no frontend app in this repo and no Node tooling in `devenv.nix`.
  Evidence: `rg --files -g 'package.json' -g 'vite.config.*' -g 'tsconfig.json' -g 'web/**' -g 'frontend/**' -g 'ui/**'` returned no files, and `devenv.nix` lists Go/audio/IaC tools but not `nodejs` or a package manager.

## Decision Log

- Decision: The front of the review card is the Chinese HelloChinese sentence with the target word highlighted; English meaning and sentence meaning appear after reveal.
  Rationale: This keeps the prompt close to the original material, avoids dictionary-card feel, and still treats the word as the review unit.
  Date/Author: 2026-04-26 / Codex

- Decision: The implementation trusts the imported file order, and also stores `index` when present. The queue orders by a stored `learning_order` integer assigned during import.
  Rationale: Josh is not fully sure the scraper can always provide reliable source position, but this full file is sorted and complete. `learning_order` is the engine's chosen order, independent from any future source metadata improvements.
  Date/Author: 2026-04-26 / Codex

- Decision: Use SQLite, but keep the new schema small: one corpus table, one cram state table, and reuse `review_events` or add one small cram log only if tests show the generic event payload is awkward.
  Rationale: JSON files felt flaky, but the earlier multi-table schema was too complicated. SQLite gives durability and constraints without forcing words, sentences, audio, decks, and graph concepts into separate tables.
  Date/Author: 2026-04-26 / Codex

- Decision: Generate canonical sentence audio on import by default, but implement a dry-run/check flow first and make the import idempotent.
  Rationale: Audio matters for the MVP and should autoplay when cards appear. Generating on import makes review fast, but import must be safe to rerun and testable even when `edge-tts` is unavailable.
  Date/Author: 2026-04-26 / Codex

- Decision: Do not add a new `hellochinese audio-smoke` command.
  Rationale: The existing public `xuezh audio tts` command already proves the exact TTS toolchain and artifact-writing path. Adding a second smoke command would expand the public contract without hiding meaningful complexity. The bulk import instructions should require a one-sentence `audio tts` smoke test before `--audio sentence`.
  Date/Author: 2026-04-26 / Codex

- Decision: Keep the new review surface separate from existing `review start` and `review grade`.
  Rationale: Existing review commands are part of the public contract and use generic numeric recall/pronunciation grades. The cram flow needs word-specific cards and `again|hard|good|easy`. A separate `cram` namespace avoids changing existing behavior and reduces risk to the current tool.
  Date/Author: 2026-04-26 / Codex

- Decision: No unknown-word graph in the first implementation.
  Rationale: The normalized HelloChinese rows are already in course order, and the first goal is to blast through the first pass. A prerequisite graph can come later after the review loop proves useful.
  Date/Author: 2026-04-26 / Codex

- Decision: Store the original HelloChinese sentence for audit/debug, but use a cleaned no-space Chinese sentence as the canonical sentence for UI and TTS.
  Rationale: The raw scraped sentence may contain spaces between Hanzi, which makes Edge TTS sound unnatural. The app should not carry two user-facing sentence concepts; `sentence_hanzi` is the cleaned canonical field, and `sentence_hanzi_raw` is only provenance.
  Date/Author: 2026-04-26 / Codex

- Decision: Generate a small fixed set of four standard Mandarin sentence audio variants, not every available voice.
  Rationale: Multiple voices reduce overfitting to one synthetic speaker, but generating every `zh-CN` voice multiplies import time and storage. Use stable non-dialect `zh-CN` voices: `zh-CN-XiaoxiaoNeural`, `zh-CN-XiaoyiNeural`, `zh-CN-YunxiNeural`, and `zh-CN-YunyangNeural`. Avoid regional voices like `zh-CN-sichuan-*`, `zh-CN-shandong-*`, `zh-CN-liaoning-*`, `zh-CN-shaanxi-*`, `zh-CN-guangxi-*`, and experimental `DragonHD*` names for the MVP.
  Date/Author: 2026-04-26 / Codex

- Decision: The web UI should be a minimal Vite + React + TypeScript app, served locally by `xuezh web serve` after the CLI engine works.
  Rationale: A single static HTML string is too weak for the actual review experience: Josh needs a fast, polished card UI with visible keyboard shortcuts, audio state, progress, and responsive layout. Vite + React is enough for that without adding auth, routing, a state library, Tailwind, a component framework, or a separate product surface. The backend remains the source of truth; the frontend is a thin client over local JSON endpoints.
  Date/Author: 2026-04-26 / Codex

- Decision: Use the `frontend-design` skill guidance for the web layer, not a broad Build Web Apps scaffold.
  Rationale: The goal is a focused review interface, not a general web app. The design bar is high enough to use real frontend structure and polish, but the implementation should stay one screen with explicit states and keyboard help.
  Date/Author: 2026-04-26 / Codex

## Outcomes & Retrospective

Implementation is complete for the intended MVP slice. The CLI imports HelloChinese JSONL rows, normalizes spaced Hanzi sentences, stores per-item cram state, returns next cards, and applies the simple `again|hard|good|easy` schedule. The web wrapper serves a one-screen React review UI through `xuezh web serve`, with visible shortcuts, reveal-before-grade flow, and local API calls into the same engine package.

## Context and Orientation

This repo is `xuezh`, a local Chinese learning engine. It is implemented in Go. The main CLI switch lives in `internal/xuezh/cli/cli.go`; the executable entry point is `cmd/xuezh-go/main.go`. SQLite database initialization and migrations are handled by `internal/xuezh/db/db.go`, `internal/xuezh/migrations/migrations.go`, and SQL files under `migrations/`. Audio generation is already implemented in `internal/xuezh/audio/audio.go` through `audio.TTSAudio`, which uses `edge-tts` and `ffmpeg`. Existing generic spaced-repetition state is in `internal/xuezh/srs/srs.go`.

Define the terms used in this plan:

`HelloChinese corpus` means the normalized JSONL file where each line is one object from the HelloChinese app. In the current file, each row has fields such as `index`, `unit_label`, `pinyin`, `hanzi`, `meaning`, `sentence_pinyin`, `sentence_hanzi`, and `sentence_meaning`.

`Review item` means one imported HelloChinese row. The target word is the row's `hanzi`; the prompt sentence is the row's `sentence_hanzi`.

`Cram mode` means fast first-pass review intended for hundreds of items in a short period. It uses short delays for missed items instead of long spaced-repetition intervals.

`Maintenance mode` means slower review after the first pass. This plan only needs enough structure to avoid blocking maintenance later; it does not need a sophisticated scheduler now.

`Autoplay` means the web page starts playing the sentence audio as soon as a new card appears. CLI output cannot autoplay by itself, so CLI returns an audio artifact path that the web layer can play.

`Keyboard shortcut` means a key the user can press instead of clicking. Shortcuts must be visible on screen using keycap-style labels, not hidden in docs. The MVP shortcuts are Space to reveal or continue, R to replay audio, and 1 through 4 to grade Again, Hard, Good, Easy.

The current complexity is that a caller would need to coordinate several generic commands: import data through `dataset import`, create or infer word IDs, generate TTS audio separately, call `review start`, call `review grade`, and know how to format the card. This work moves that sequencing into a purpose-built HelloChinese cram module. Callers should only need to ask for the next card and grade it.

## Plan of Work

First, add a migration that stores imported HelloChinese rows and their cram state. Create `migrations/0005_add_hellochinese_cram.sql`. Add a table named `hellochinese_items` with a UUID `id`, `learning_order integer not null unique`, source fields, target fields, prompt sentence fields, `row_hash text not null unique`, one `sentence_audio_paths_json text` column, and timestamps. `sentence_audio_paths_json` is a JSON object whose keys are voice IDs and whose values are workspace-relative Ogg paths. Store `sentence_hanzi` as the cleaned no-space canonical Chinese sentence, and store `sentence_hanzi_raw` only for provenance/debug if it differs. Add a table named `hellochinese_cram_state` with `item_id` as primary key, `status`, `next_due_at`, `seen_count`, `lapse_count`, `last_grade`, and timestamps. Reuse the existing `review_events` table for grade logs by writing payload JSON with `mode: "hellochinese_cram"`; do not add a separate cram log table in this MVP.

Second, create a focused package at `internal/xuezh/hellochinese/`. This package should hide import, audio generation, card selection, and grading policy from the CLI. Use plain structs such as `ImportOptions`, `ImportResult`, `Card`, `GradeResult`, and functions such as `ImportCorpus`, `NextCards`, and `GradeCard`. The package should parse the JSONL file line by line so large files do not require loading all rows at once. During normalization, remove artificial spaces between Chinese characters before storing `sentence_hanzi`; keep the original value in `sentence_hanzi_raw`. For each row, compute a `row_hash` from canonical JSON for the inserted fields. Generate a UUID using `github.com/google/uuid`, which is already in `go.mod`. Assign `learning_order` from the row's numeric `index` if present and complete; otherwise assign the JSONL line number. For this corpus, `index` is complete and should become `learning_order`.

Third, implement idempotent import. If the same `row_hash` already exists, leave the row alone. If the same `learning_order` exists with a different `row_hash`, return a typed conflict error instead of silently overwriting content. If the same `word + sentence` appears again with different order, also report a conflict. On successful insert, create a `hellochinese_cram_state` row with `status = "new"`. The import result should include `rows_seen`, `rows_inserted`, `rows_existing`, `audio_generated`, `audio_existing`, and `audio_failed`. Use existing error types from `internal/xuezh/errors/errors.go`; for conflicts caused by input data, use `INVALID_ARGUMENT` unless a new error type is deliberately added across the contract tests.

Fourth, require an audio smoke gate before bulk audio generation, using the existing public TTS command:

    xuezh audio tts --text "你是龙大。" --voice zh-CN-XiaoxiaoNeural --out artifacts/smoke/sentence.ogg --backend edge-tts --json

This command already calls `audio.TTSAudio`, which is the same path import will use. It must succeed in the same runtime environment before any 689-row audio generation is attempted. If it fails with the existing `TOOL_MISSING` error, fix the environment or run import with `--audio none`.

Fifth, implement sentence audio preparation. Add an import flag such as `--audio sentence|none` and default it to `none` until the smoke command succeeds. After smoke passes, the intended production command is `--audio sentence`. Add a `--voices` flag with a conservative default of `zh-CN-XiaoxiaoNeural,zh-CN-XiaoyiNeural,zh-CN-YunxiNeural,zh-CN-YunyangNeural`. For each imported or existing row missing audio for a requested voice, call `audio.TTSAudio(row.SentenceHanzi, voice, outPath, "edge-tts", "hellochinese_sentence_tts")`. The text passed to TTS must be the cleaned `sentence_hanzi`, never `sentence_hanzi_raw`. The output path should be deterministic and workspace-relative, for example `artifacts/hellochinese/sentences/<item-id>/<voice>.ogg`. If audio generation fails because `edge-tts` or `ffmpeg` is missing, import should fail clearly unless the user passes `--audio none`. The first implementation should not generate word audio unless it falls out naturally; sentence audio is enough because the web card autoplays the sentence.

Sixth, add CLI commands in `internal/xuezh/cli/cli.go`. Add a top-level `hellochinese` command for corpus/audio preparation and a top-level `cram` command for review. Do not change the behavior of existing `review start`, `review grade`, `audio tts`, or `dataset import`; those commands are already in `specs/cli/contract.json` and should remain stable.

    xuezh hellochinese import --path /path/to/hellochinese_words.jsonl --audio sentence --json
    xuezh cram next --limit 1 --json
    xuezh cram grade --item <uuid> --grade again|hard|good|easy --json

The `cram next` output should return cards with `item_id`, `learning_order`, `word`, `pinyin`, `meaning`, `sentence_hanzi`, `sentence_pinyin`, `sentence_meaning`, `sentence_audio_paths`, `status`, `due_at`, and `unknown_other_words`. `sentence_audio_paths` is a map from voice ID to workspace-relative audio path. The engine should not randomly select a voice; that would make CLI output less deterministic. The tiny web page can pick a random available voice in browser JavaScript when it autoplays the card. For this first version, `unknown_other_words` can be `null` or `0` because no prerequisite graph is being built yet. Do not expose recommendation fields such as `recommended_next`.

Seventh, implement the simple schedule. For `cram grade`, map grades to next due times:

    again -> now
    hard  -> now + 10 minutes
    good  -> now + 2 hours
    easy  -> now + 24 hours

Increment `seen_count` on every grade. Increment `lapse_count` for `again`. Set `status` to `learning` after the first grade unless the item is `easy`, in which case `status` can be `review`. This is deliberately simple and should live in one function inside `internal/xuezh/hellochinese/`, not spread through CLI code.

Eighth, add tests. This repo currently has no checked-in Go tests, so start with package-level Go tests next to the new package rather than inventing a separate test harness. Unit tests should cover JSONL parsing, row hashing idempotence, conflict detection, next-card ordering, and grade scheduling. Integration-style tests should use a temporary workspace and a small fixture JSONL with three rows under `internal/xuezh/hellochinese/testdata/`. To avoid depending on real `edge-tts` in tests, use `--audio none` in most tests and inject a fake audio generator into the package for audio-path behavior. Add one command-level test only if the existing CLI structure makes it straightforward; otherwise validate the package functions and keep CLI code thin.

Ninth, update the contract artifacts only for commands that become public. The repo's contract authority says public CLI changes require updates to `docs/cli-contract.md`, `specs/cli/contract.json`, `schemas/`, `specs/bdd/`, and contract checks if they exist. There is no `tests/` directory in this working tree, so do not reference files that do not exist; instead, update the existing contract artifacts and add Go tests or BDD feature files in the style already present under `specs/bdd/`. Add schemas for `hellochinese.import`, `cram.next`, and `cram.grade`. Add BDD scenarios that import a tiny HelloChinese fixture with `--audio none`, fetch the first card, and grade it.

Tenth, build the mini web app only after the CLI is working. Add minimal Node tooling to `devenv.nix` such as `nodejs_22` and `pnpm`, then create a small frontend under `web/` using Vite + React + TypeScript. Keep the frontend deliberately narrow: one route, one card screen, no router, no global state library, no Tailwind, no component framework, no auth, no cloud calls. Add `xuezh web serve --port 8765` in Go. In development it can proxy to the Vite dev server if that is simpler; for the handoff it should serve the built Vite assets from the Go binary or from a checked-in build directory only if embedding is too much for the first pass. The web server should call the same package functions, not shell out to the CLI. It should serve JSON endpoints: `GET /api/cram/next`, `POST /api/cram/grade`, and `GET /artifacts/...` for workspace audio files.

The card page should be designed as a real daily-use tool, following the `frontend-design` skill guidance. The primary element is the Chinese sentence, large and centered, with the target word highlighted when exact-once matching works. The secondary elements are progress, current voice, replay state, and the reveal/grade controls. Keyboard shortcuts must be shown on screen in the control labels: `Space` Reveal / Next, `R` Replay, `1` Again, `2` Hard, `3` Good, `4` Easy. The app needs loading, empty, and error states; visible focus states; 44px minimum click targets; and no hover-only controls. The visual style should be quiet and study-focused, not a marketing page: dense enough for repeated use, high contrast, restrained color, and stable layout with no card-inside-card clutter.

This plan intentionally does not add a tutor, chat mode, Anki export, iOS app, accounts, cloud sync, deck editor, generated content, prerequisite graph, or sophisticated SRS. Those can be added later if the cram loop is useful.

## Concrete Steps

Run all commands from `/Users/josh/code/xuezh`.

Start by checking the current tree:

    git status --short

Add the migration:

    migrations/0005_add_hellochinese_cram.sql

Create the package:

    internal/xuezh/hellochinese/

Expected new Go files are:

    internal/xuezh/hellochinese/import.go
    internal/xuezh/hellochinese/review.go
    internal/xuezh/hellochinese/audio.go
    internal/xuezh/hellochinese/import_test.go
    internal/xuezh/hellochinese/review_test.go

Wire CLI commands in:

    internal/xuezh/cli/cli.go

Add public contract files only after command names are final:

    schemas/hellochinese.import.schema.json
    schemas/cram.next.schema.json
    schemas/cram.grade.schema.json
    specs/bdd/hellochinese.feature
    docs/cli-contract.md
    specs/cli/contract.json

When the web milestone starts, add the minimal frontend files:

    web/package.json
    web/vite.config.ts
    web/tsconfig.json
    web/src/main.tsx
    web/src/App.tsx
    web/src/styles.css

Also update:

    devenv.nix

to include the Node/package-manager tools needed to run and build the web app. Do not add global installs.

Use this real corpus for manual verification, but do not modify it:

    /Users/josh/Documents/Codex/2026-04-24/can-you-use-the-hellochinese-plugin/snapshots/hellochinese_words.jsonl

Before implementing bulk audio import, verify the current audio reality. This command fails in the normal shell because `edge-tts` is missing from `PATH`; that failure is expected outside `devenv`:

    XUEZH_WORKSPACE_DIR=/tmp/xuezh-audio-smoke go run ./cmd/xuezh-go audio tts --text '你 是 龙大。' --voice XiaoxiaoNeural --out artifacts/smoke/sentence.ogg --backend edge-tts --json

Expected outside-devenv failure:

    "ok": false
    "command": "audio.tts"
    "error": {
      "type": "TOOL_MISSING",
      "message": "Required tool not found on PATH: edge-tts"
    }

Use the repo `devenv`, not a global install. The correct command form is `devenv shell sh -lc '<command>'`; do not use `devenv shell -- <command>` for this repo. Also prefer `/private/tmp/...` over `/tmp/...` for temporary workspaces on macOS because the path safety code compares resolved paths.

Before bulk audio import, verify one-sentence audio through the same path import will use:

    devenv shell sh -lc 'XUEZH_WORKSPACE_DIR=/private/tmp/xuezh-audio-smoke go run ./cmd/xuezh-go audio tts --text "你是龙大。" --voice zh-CN-XiaoxiaoNeural --out artifacts/smoke/sentence.ogg --backend edge-tts --json'

Only after this succeeds should a bulk import use `--audio sentence`.

After implementation, verify CLI behavior without audio first:

    devenv shell sh -lc 'XUEZH_WORKSPACE_DIR=/private/tmp/xuezh-cram-smoke go run ./cmd/xuezh-go db init --json'
    devenv shell sh -lc 'XUEZH_WORKSPACE_DIR=/private/tmp/xuezh-cram-smoke go run ./cmd/xuezh-go hellochinese import --path /Users/josh/Documents/Codex/2026-04-24/can-you-use-the-hellochinese-plugin/snapshots/hellochinese_words.jsonl --audio none --json'
    devenv shell sh -lc 'XUEZH_WORKSPACE_DIR=/private/tmp/xuezh-cram-smoke go run ./cmd/xuezh-go cram next --limit 1 --json'

The first card should be the first HelloChinese row. It should include the cleaned Chinese sentence `你是龙大。`, target word `你`, pinyin `nǐ`, and English answer `you (singular)`.

Then test grading:

    devenv shell sh -lc 'XUEZH_WORKSPACE_DIR=/private/tmp/xuezh-cram-smoke go run ./cmd/xuezh-go cram grade --item <item-id-from-next> --grade good --json'
    devenv shell sh -lc 'XUEZH_WORKSPACE_DIR=/private/tmp/xuezh-cram-smoke go run ./cmd/xuezh-go cram next --limit 1 --json'

The graded item should no longer be immediately returned if another new item is available, because `good` schedules it two hours later.

When audio support is wired and the smoke command succeeds, verify a small import with real audio:

    devenv shell sh -lc 'XUEZH_WORKSPACE_DIR=/private/tmp/xuezh-cram-audio go run ./cmd/xuezh-go hellochinese import --path internal/xuezh/hellochinese/testdata/min.jsonl --audio sentence --json'

Expect one artifact path per requested voice under the workspace such as:

    artifacts/hellochinese/sentences/<uuid>/zh-CN-XiaoxiaoNeural.ogg

When the web app exists, verify it in development:

    devenv shell sh -lc 'cd web && pnpm install && pnpm dev --host 127.0.0.1'

Then start the Go API/server in another shell if the implementation splits dev frontend and backend:

    devenv shell sh -lc 'XUEZH_WORKSPACE_DIR=/private/tmp/xuezh-cram-smoke go run ./cmd/xuezh-go web serve --port 8765'

For final verification, build and serve through Go:

    devenv shell sh -lc 'cd web && pnpm build'
    devenv shell sh -lc 'XUEZH_WORKSPACE_DIR=/private/tmp/xuezh-cram-smoke go run ./cmd/xuezh-go web serve --port 8765'

Open `http://localhost:8765` and verify the review screen is usable without reading external docs.

Run the full gate:

    ./scripts/check.sh

It should run `go test ./...` and pass.

## Validation and Acceptance

Acceptance is user-visible behavior, not just compilation.

Given a normalized HelloChinese JSONL file with three rows, when `xuezh hellochinese import --path <fixture> --audio none --json` runs, then the command returns `ok: true`, reports three rows seen, and stores three reviewable items in learning order.

Given the imported fixture, when `xuezh cram next --limit 1 --json` runs before any grades, then it returns the first row by `learning_order` with the Chinese sentence on the front, the target word, pinyin, English meaning, sentence meaning, and no recommendation fields.

Given the first card is graded `good`, when `xuezh cram grade --item <id> --grade good --json` runs, then the returned `next_due_at` is about two hours after the grade time and a following `cram next` returns the next unreviewed row.

Given the real 689-row corpus, when import runs with `--audio none`, then it imports without modifying the source file and returns a row count of 689.

Given the command is run outside the repo `devenv`, when `xuezh audio tts --text "你是龙大。" --voice zh-CN-XiaoxiaoNeural --out artifacts/smoke/sentence.ogg --backend edge-tts --json` runs, then it fails with `TOOL_MISSING` and does not create bulk audio artifacts. Given the same command is run through `devenv shell sh -lc` with `XUEZH_WORKSPACE_DIR=/private/tmp/...`, then it succeeds and creates one Ogg/Opus sentence artifact. This proves the runtime is ready before any 689-row generation job starts without adding a second smoke-test command.

Given `edge-tts` and `ffmpeg` are available, when import runs with `--audio sentence` on a tiny fixture, then each imported row has a sentence audio path and that file exists under the workspace.

Given the web server is started after CLI implementation, when the user opens `http://localhost:8765`, then the first card shows the Chinese sentence with the target highlighted, autoplays sentence audio if available, shows visible shortcut labels for Space, R, and 1 through 4, reveals the English answer on Space, replays audio on R, and grades with keys 1 through 4.

Given the card has not been revealed, then grade buttons are present but visually secondary or disabled until reveal. Given the answer is revealed, then the grade buttons become the primary action row and Space advances only after a grade has been recorded. This prevents accidental grading while keeping the flow fast.

Given the web app is loading, has no imported items, or receives an API error, then it shows a specific loading, empty, or error state with a recovery action. It must not show a blank page.

Given the user tabs through the web app, then focus is visible on every interactive control and all primary actions are keyboard-accessible.

Run `./scripts/check.sh` before handoff and expect all Go tests to pass.

## Idempotence and Recovery

Import must be safe to rerun. A row with the same `row_hash` should be counted as existing, not inserted again. A row with the same `learning_order` but different content should stop with a clear conflict error so the user can inspect the normalized file. The source JSONL file must never be modified.

Audio generation must be safe to resume. If a row already has audio for a requested voice and the file exists, import should not regenerate it unless a future explicit `--force-audio` flag is added. Do not add that flag in this MVP. Bulk audio generation must be gated by a successful one-sentence `audio tts` smoke test; this protects the user from creating hundreds of failed subprocess attempts or partial artifacts without expanding the public CLI surface.

If real audio tools are unavailable, the user can run import with `--audio none` and still test the review loop. Tests should not depend on network access or external TTS unless explicitly marked as an integration smoke test. If `devenv shell` appears to hang, first verify the command form: use `devenv shell sh -lc '...'`, not `devenv shell -- ...`. Do not work around missing tools with a global pip install; use the already declared Nix package through the repo's environment.

If web work causes trouble, stop after the CLI milestone. The CLI is the architectural boundary and the web layer is only a convenience wrapper.

If frontend dependency installation fails, do not replace the design with a static HTML string. Fix the repo-local `devenv.nix` and `web/package.json` setup, or temporarily continue with CLI verification while leaving the web milestone incomplete in this ExecPlan.

## Artifacts and Notes

Corpus inspection evidence:

    $ wc -l hellochinese_words.jsonl
    689 hellochinese_words.jsonl

    $ jq index check
    min 1 max 689 present 689 missing 0

Example first rows from the real corpus:

    index=1 word=你 pinyin=nǐ meaning="you (singular)" raw sentence="你 是 龙大。" cleaned sentence="你是龙大。"
    index=2 word=我 pinyin=wǒ meaning="I; me" raw sentence="我 是 龙大。" cleaned sentence="我是龙大。"
    index=3 word=是 pinyin=shì meaning="to be" raw sentence="我 是 龙大。" cleaned sentence="我是龙大。"

Expected card front in the web UI:

    你是龙大。

with `你` highlighted. Expected reveal:

    nǐ
    you (singular)
    You (singular) are Long Da.

Expected visible shortcut labels in the web UI:

    Space Reveal
    R Replay
    1 Again
    2 Hard
    3 Good
    4 Easy

Audio smoke evidence from this planning pass:

    $ XUEZH_WORKSPACE_DIR=/tmp/xuezh-audio-smoke go run ./cmd/xuezh-go audio tts --text '你 是 龙大。' --voice XiaoxiaoNeural --out artifacts/smoke/sentence.ogg --backend edge-tts --json
    {
      "ok": false,
      "command": "audio.tts",
      "error": {
        "type": "TOOL_MISSING",
        "message": "Required tool not found on PATH: edge-tts"
      }
    }

Working devenv audio smoke evidence:

    $ devenv shell sh -lc 'command -v edge-tts; command -v ffmpeg'
    /nix/store/...-python3.13-edge-tts-7.2.3/bin/edge-tts
    /nix/store/...-ffmpeg-8.0-bin/bin/ffmpeg

    $ devenv shell sh -lc 'XUEZH_WORKSPACE_DIR=/private/tmp/xuezh-audio-smoke-devenv go run ./cmd/xuezh-go audio tts --text "你是龙大。" --voice XiaoxiaoNeural --out artifacts/smoke/sentence.ogg --backend edge-tts --json'
    {
      "ok": true,
      "command": "audio.tts",
      "data": {
        "out": "artifacts/smoke/sentence.ogg",
        "text": "你是龙大。",
        "voice": "zh-CN-XiaoxiaoNeural"
      },
      "artifacts": [
        {
          "path": "artifacts/smoke/sentence.ogg",
          "mime": "audio/ogg",
          "purpose": "tts_audio",
          "bytes": 3959
        }
      ]
    }

    $ file /private/tmp/xuezh-audio-smoke-devenv/artifacts/smoke/sentence.ogg
    Ogg data, Opus audio, version 0.1, mono, 48000 Hz

    $ ffprobe -v error -show_entries format=duration,size -of default=noprint_wrappers=1 /private/tmp/xuezh-tts-nospace/artifacts/samples/sample-1.ogg
    duration=1.494500
    size=3959

## Interfaces and Dependencies

In `internal/xuezh/hellochinese/import.go`, define small data types that keep CLI code thin:

    type ImportOptions struct {
        Path string
        AudioMode string
        Voices []string
    }

    type ImportResult struct {
        RowsSeen int
        RowsInserted int
        RowsExisting int
        AudioGenerated int
        AudioExisting int
        AudioFailed int
    }

    func ImportCorpus(opts ImportOptions) (ImportResult, error)

`ImportCorpus` hides JSONL parsing, order assignment, row hashing, idempotent insert, state initialization, and optional audio generation. Callers should not need to know the table names or TTS output convention.

In `internal/xuezh/hellochinese/review.go`, define:

    type Card struct {
        ItemID string
        LearningOrder int
        Word string
        Pinyin string
        Meaning string
        SentenceHanzi string
        SentencePinyin string
        SentenceMeaning string
        SentenceAudioPaths map[string]string
        Status string
        DueAt *string
    }

    func NextCards(limit int, now time.Time) ([]Card, error)
    func GradeCard(itemID string, grade string, now time.Time) (GradeResult, error)

`NextCards` hides the queue policy: due learning items first, then new items in learning order. `GradeCard` hides the grade-to-delay mapping.

In `internal/xuezh/cli/cli.go`, the CLI should only parse flags, call these package functions, and emit envelopes. Do not put scheduling or import policy in the CLI.

For web, use Go's standard `net/http` for the local API and Vite + React + TypeScript for the browser UI. The web server should call `hellochinese.NextCards` and `hellochinese.GradeCard` directly. The frontend should call local endpoints only; it should not know SQLite, TTS, scheduling, or import details. This keeps future agents and iOS clients free to reuse the same engine behavior through either CLI JSON or the minimal HTTP surface.

Revision note: This initial ExecPlan records the agreed product direction: Chinese sentence prompt with target highlight, English after reveal, sentence audio autoplay in the web UI, SQLite with a small schema, import-time sentence audio generation, file order as learning order, CLI first, tiny web second.

Revision note: Improved on 2026-04-26 after code-grounded audit and audio smoke testing. Added a one-sentence audio smoke gate using the existing `audio tts` command, documented that `edge-tts` is missing outside `devenv` but works inside the repo `devenv`, recorded the `/tmp` versus `/private/tmp` path-safety gotcha, corrected test guidance because no `tests/` directory currently exists, and made the namespace decision explicit so the existing public `review` commands are not disturbed.

Revision note: Updated on 2026-04-26 after TTS sample testing showed spaced Hanzi sounds unnatural. The plan now makes cleaned no-space `sentence_hanzi` canonical, preserves raw sentence text only for provenance, generates audio only from cleaned text, and supports four standard Mandarin voices with randomized playback.

Revision note: Improved on 2026-04-26 for simplicity. Removed the proposed `hellochinese audio-smoke` command because existing `audio tts` already provides the same proof, moved random voice selection out of the engine and into the tiny web page, and represented per-voice audio paths as one JSON column instead of a new table.

Revision note: Updated on 2026-04-26 after applying `frontend-design` guidance and reviewing the repo's lack of frontend tooling. The plan now calls for CLI-first implementation followed by a minimal Vite + React + TypeScript single-screen app with visible keyboard shortcuts and complete loading/empty/error states. This replaces the too-simple static HTML wrapper while still avoiding a broad Build Web Apps scaffold, routing, Tailwind, auth, cloud sync, or a separate product architecture.
