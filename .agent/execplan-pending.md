# Make Xuezh Whole-Deck Offline on iPhone

This ExecPlan is a living document. The sections `Progress`, `Surprises & Discoveries`, `Decision Log`, and `Outcomes & Retrospective` must be kept up to date as work proceeds.

This repository has `.agent/PLANS.md`; maintain this document according to that file.

## Purpose / Big Picture

After this change, Josh can install the local xuezh web app as an iPhone Home Screen app, press one explicit control to make the whole cram deck available offline, get on a plane with no internet, choose cards from the offline copy, review sentences with audio, reload or reopen the app, and safely resume. When the phone is online again, xuezh sends the offline answers back to the Mac server and applies them to the real SQLite score state exactly once.

The current app already has the right server-side truth: `internal/xuezh/cram` owns canonical cards, Pleco-style scoring, due dates, review sessions, undo, and event logging. The missing boundary is an offline boundary. Today the browser must call `/api/cram/...` for every meaningful action. That leaks network availability into the review loop and makes a plane session impossible. This plan adds one deep boundary: a whole-deck offline snapshot plus an append-only answer event log. The phone may cache and replay a temporary view of the deck, but Go remains the scoring authority when events sync.

## Progress

- [x] (2026-05-05T10:38:03Z) Inspected the current session, scoring, query, webserver, and React review paths.
- [x] (2026-05-05T10:38:03Z) Confirmed the repo tree is clean before starting.
- [x] (2026-05-05T10:38:03Z) Wrote this ExecPlan.
- [x] (2026-05-05T20:18Z) Add server offline deck export and idempotent offline event sync.
- [x] (2026-05-05T20:18Z) Add PWA manifest and service worker for app-shell and audio caching.
- [x] (2026-05-05T20:18Z) Add browser IndexedDB storage for the offline deck, active offline session, and pending events.
- [x] (2026-05-05T20:18Z) Wire offline status, whole-deck download, offline session creation, offline grading, and foreground sync into the existing React app.
- [x] (2026-05-05T20:31Z) Added Go coverage for offline deck export, ordered event sync, duplicate-event skip, and live score updates.
- [x] (2026-05-05T20:44Z) Ran a clean throwaway browser smoke: saved the deck offline, stopped the server, reviewed two cards offline, reloaded offline, restarted the server, synced, and verified exactly two `review_events`.
- [x] (2026-05-05T12:09Z) Ran an active-offline-session smoke with the server returning mid-session; verified no sync happens until the local session finishes, then all queued events apply once.
- [x] (2026-05-05T12:09Z) Ran visual regression checks at 393px phone width and desktop width; fixed picker horizontal overflow and confirmed `scrollWidth == clientWidth`.
- [x] (2026-05-05T12:09Z) Ran `devenv shell -- ./scripts/check.sh` and `devenv shell -- sh -lc 'cd web && pnpm build'`.
- [x] (2026-05-05T12:14Z) Reviewed for simplicity, removed unnecessary drift risk, updated operating docs, and prepared one clean slice for commit.

## Surprises & Discoveries

- Observation: The local cram workspace is small enough for whole-deck offline.
  Evidence: A prior probe in this thread showed `/Users/josh/.local/share/xuezh/cram-local` is about 87M, with audio artifacts about 85M and roughly 10,000 audio files.

- Observation: The server already stores enough review event payload to undo and replay scoring changes.
  Evidence: `internal/xuezh/cram/scheduler.go` writes `review_events` with old and new score snapshots; `internal/xuezh/cram/session.go` already grades inside a transaction through `gradeCardTx`.

- Observation: The current browser app has no offline branch.
  Evidence: `web/src/main.tsx` fetches `/api/cram/overview`, `/api/cram/preview`, `/api/cram/session`, and review endpoints directly. If those fetches fail, the app can only show an error.

- Observation: Browser offline sync must be tested with the server genuinely stopped, not just by forcing an app flag.
  Evidence: In the clean smoke workspace, two answers were made after killing the `18766` server; after restart, SQLite showed exactly two `review_events`, `你` at score `100` with one incorrect review, and `我` at score `600` with one correct review.

- Observation: The phone picker had a real horizontal overflow regression after the offline status control was added.
  Evidence: A 393px browser screenshot clipped the right side of the filter/footer controls. After changing mobile grid tracks to `minmax(0, 1fr)` and stacking offline status/action, CDP metrics reported `scrollWidth=393` and `clientWidth=393`.

## Decision Log

- Decision: Whole deck offline is the target, not “save this 100-card round.”
  Rationale: The full deck plus audio is small enough, and the user needs to choose filters/categories while offline.
  Date/Author: 2026-05-05 / Codex

- Decision: Offline answers are append-only facts.
  Rationale: Reviewing the same card twice is valid. There is no conflict UI. The server applies events by answered time and event ID, skipping duplicates.
  Date/Author: 2026-05-05 / Codex

- Decision: The phone keeps a temporary read model; the Go server remains scoring authority.
  Rationale: This avoids duplicating the scheduler as a second product. The offline UI can update local session progress immediately, but durable score truth is written by Go during sync.
  Date/Author: 2026-05-05 / Codex

- Decision: Use PWA first, not native iOS.
  Rationale: A PWA can cache the app shell, IndexedDB data, and audio without a SwiftUI rewrite. Native iOS remains a fallback only if real iPhone Home Screen testing shows storage or audio cannot be trusted.
  Date/Author: 2026-05-05 / Codex

## Outcomes & Retrospective

Implementation is complete. The app now has a whole-deck offline snapshot, service-worker app shell, IndexedDB card/session/event storage, offline review progression, and idempotent foreground sync back to SQLite. The smoke proof uses disposable workspaces and does not mutate Josh's real card scores. Visual QA caught and fixed one mobile overflow regression before commit.

## Context and Orientation

The repo is `/Users/josh/code/xuezh`. The backend is Go. The main local web server is `internal/xuezh/webserver/webserver.go`. It serves the built Vite app from `web/dist`, exposes `/api/cram/...` JSON endpoints, and serves generated audio files under `/artifacts/...` from the configured workspace.

The cram domain code lives in `internal/xuezh/cram`. `types.go` defines cards, practice previews, grade options, and review session state. `query.go` reads the canonical card and score state from SQLite. `session.go` owns review session progression. `scheduler.go` applies Pleco-style `correct|incorrect` scoring and writes `review_events`. `session_store.go` persists active review session queues.

The React app lives under `web/src`. `main.tsx` coordinates overview, practice preview, review session, audio playback, and keyboard shortcuts. `BatchPicker.tsx` renders the “What to review” screen. `ReviewSession.tsx` renders the flashcard and session footer. The design-system/debug surface is intentionally isolated in `web/src/dev`.

Terms used in this plan:

An `offline deck snapshot` is one JSON payload containing every card, its current score facts, category/source summaries, and all audio paths. It is a read model: the phone uses it to select cards and render review while offline.

An `offline answer event` is one immutable local fact: event ID, session ID, item ID, `correct` or `incorrect`, shown time, answer time, elapsed milliseconds, round, and retry flag. Events are queued on the phone and later sent to the server.

`IndexedDB` is the browser database used for durable structured data on the iPhone. The app uses it for cards, session state, and pending answer events.

`Cache Storage` is the service-worker cache used for static assets and audio responses.

## Plan of Work

First, add backend offline export and sync in `internal/xuezh/cram/offline.go`. Export must return the whole deck in one server-owned shape: cards, scores, due dates, category/source summaries, scoring settings needed for local filtering, and a flattened list of audio paths. Sync must accept offline answer events, sort them by answered time then event ID, skip already-applied event IDs, and apply each missing event in one SQLite transaction through the existing `gradeCardTx`. Add `EventID` to `GradeOptions` so the sync path can preserve phone-generated event IDs and retry safely. Do not add a second scoring algorithm in Go.

Second, add webserver endpoints in `internal/xuezh/webserver/webserver.go`: `GET /api/cram/offline/deck`, `POST /api/cram/offline/sync`, and `GET /offline/app-shell`. The app-shell endpoint should read `web/dist/index.html`, find the current built asset URLs, and return the static URLs that the service worker should cache. Keep this endpoint small; it exists only so the service worker does not need to know Vite hash names.

Third, add PWA files. Add `web/public/manifest.webmanifest` and `web/public/sw.js`, and link the manifest in `web/index.html`. The service worker should cache the app shell on install/activation and expose messages for caching offline audio URLs. Keep it direct: cache-first for same-origin app assets and artifacts, network-first for API except when offline data is intentionally loaded from IndexedDB by React. Do not add Workbox or another dependency unless a direct service worker proves insufficient.

Fourth, add browser offline storage in `web/src/offline.ts`. This module hides IndexedDB and service-worker sequencing from React. It should provide a small surface: `registerOfflineApp()`, `saveOfflineDeck(snapshot, onProgress)`, `loadOfflineDeck()`, `saveOfflineSession(session)`, `loadOfflineSession()`, `appendOfflineEvent(event)`, `pendingOfflineEvents()`, and `markOfflineEventsSynced(ids)`. React should not open IndexedDB stores directly.

Fifth, wire the existing React app to use this offline boundary. `main.tsx` should still prefer the server when online. If server fetches fail and an offline deck exists, it should render the same `BatchPicker` and `ReviewCard` using locally computed preview/session state. Add one visible control on the picker: `Make available offline`, with clear states such as `Saving cards`, `Saving audio`, `Offline ready`, and `N answers to sync`. When offline, starting a review session creates a local session from selected cards and stores it in IndexedDB. Revealing, repeat-later, grading, and undo must update the local session and pending event log without network.

Sixth, sync when online in the foreground. On app start and after online actions, if pending offline events exist, send them to `/api/cram/offline/sync`. After server acknowledgement, delete only acknowledged local events, refresh the server deck snapshot, and update the offline copy. Do not rely on background sync. The UI should show sync status plainly.

Seventh, update tests and QA. Add Go tests for deck export and idempotent event sync. Add frontend build coverage through the existing `pnpm build`. Add a browser smoke using the in-app browser or local automation: build, run against a throwaway workspace, save offline deck, create an offline session, grade multiple cards while the server is unavailable or offline mode is forced, reload, confirm the session resumes, restart the server, sync, and confirm server score rows changed exactly once.

Eighth, review for simplicity. The implementation should have one offline module, not scattered IndexedDB calls. It should not add CLI knobs, duplicate schedulers, or compatibility migrations. If implementation adds too many lines, delete or flatten before handoff.

## Concrete Steps

Run from `/Users/josh/code/xuezh`.

Before editing, verify the tree:

    git status --short

After backend changes:

    gofmt -w internal/xuezh/cram internal/xuezh/webserver/webserver.go
    devenv shell -- go test ./internal/xuezh/cram ./internal/xuezh/webserver

After frontend changes:

    devenv shell -- sh -lc 'cd web && pnpm build'

For full verification:

    devenv shell -- ./scripts/check.sh

For safe smoke testing, always use a throwaway workspace:

    export XUEZH_WORKSPACE_DIR=/private/tmp/xuezh-offline-smoke
    trash "$XUEZH_WORKSPACE_DIR" 2>/dev/null || rm -rf "$XUEZH_WORKSPACE_DIR"
    devenv shell -- go run ./cmd/xuezh-go hellochinese import --path internal/xuezh/cram/testdata/hellochinese.txt --audio none --json
    devenv shell -- go run ./cmd/xuezh-go travel import --path internal/xuezh/cram/testdata/travel.txt --audio none --json
    XUEZH_SKIP_AUDIO_BACKFILL=1 devenv shell -- go run ./cmd/xuezh-go web serve --port 8765

Use the in-app browser at `http://127.0.0.1:8765/` and design-system routes. Capture mobile-width screenshots for the picker, review before reveal, review after reveal, long sentence, and long answer. Reject the UI if controls wrap, content clips, prompt position changes between reveal states, or offline controls are unclear.

## Validation and Acceptance

The backend is acceptable when a test proves that two offline events for the same or different cards can be submitted, a duplicate submission is skipped, and the score rows plus `review_events` show exactly one application per event ID.

The PWA is acceptable when the built app registers a service worker, the manifest is served, and app shell assets can be loaded from the service-worker cache after network requests are blocked in browser automation.

Whole-deck offline is acceptable when `Make available offline` saves all cards from a throwaway database, stores all existing audio paths that can be fetched, reports missing audio without generating anything, and reloads the app from local data when the server is unavailable.

Offline review is acceptable when the browser can start a session from the offline deck, reveal a card, answer `Incorrect`, answer another card `Correct`, reload the page, and resume with the same queue and pending event count.

Sync is acceptable when the server comes back, pending events sync in order, local pending event count drops to zero only after acknowledgement, and a fresh server preview shows the updated score/count/due facts.

Regression acceptance requires `devenv shell -- ./scripts/check.sh`, `devenv shell -- sh -lc 'cd web && pnpm build'`, and browser screenshots for the normal app/design-system surfaces named above.

## Idempotence and Recovery

Offline deck download is safe to rerun. It replaces the local offline snapshot with the latest server state only after the new snapshot is fully stored. Audio caching is fill-missing. It must not call TTS and must not replace existing audio on disk.

Offline event sync is safe to retry. Each offline event has a stable event ID. The server skips event IDs already present in `review_events`. If the network fails after the server commits but before the phone receives the response, the next sync sends the same events and the server skips duplicates.

Use throwaway workspaces for tests and browser smokes. Do not mutate Josh's real cram-local database except for read-only UI screenshots unless explicitly asked.

If service-worker behavior appears stale, unregister the service worker in browser dev tools or change the service-worker version string, rebuild, and reload. Do not mask stale app behavior by adding more runtime branches.

## Artifacts and Notes

Relevant current files:

    internal/xuezh/cram/types.go
    internal/xuezh/cram/query.go
    internal/xuezh/cram/session.go
    internal/xuezh/cram/session_store.go
    internal/xuezh/cram/scheduler.go
    internal/xuezh/webserver/webserver.go
    web/src/main.tsx
    web/src/BatchPicker.tsx
    web/src/ReviewSession.tsx
    web/src/types.ts
    web/src/utils.ts
    web/index.html
    web/vite.config.ts

The server is already ACID for normal review: `GradeReviewSession` opens a SQLite transaction, calls `gradeCardTx`, advances the session, saves the session, and commits. Offline sync should reuse the same lower-level scoring path rather than inventing a new scorer.

## Interfaces and Dependencies

In `internal/xuezh/cram/types.go`, add:

    type OfflineDeckSnapshot struct { ... }
    type OfflineDeckCard struct { ... }
    type OfflineReviewEvent struct { ... }
    type OfflineSyncResult struct { ... }

In `internal/xuezh/cram/offline.go`, define:

    func OfflineDeck(now time.Time) (OfflineDeckSnapshot, error)
    func SyncOfflineReviewEvents(events []OfflineReviewEvent, syncedAt time.Time) (OfflineSyncResult, error)

`OfflineDeck` hides SQL details and returns the exact read model the browser needs. `SyncOfflineReviewEvents` hides idempotency, event ordering, and transaction boundaries from the webserver.

In `internal/xuezh/webserver/webserver.go`, add:

    GET  /api/cram/offline/deck
    POST /api/cram/offline/sync
    GET  /offline/app-shell

In `web/src/offline.ts`, define:

    registerOfflineApp(): Promise<void>
    saveOfflineDeck(snapshot, progress): Promise<void>
    loadOfflineDeck(): Promise<OfflineDeckSnapshot | null>
    saveOfflineSession(session): Promise<void>
    loadOfflineSession(): Promise<ReviewSessionState | null>
    appendOfflineEvent(event): Promise<void>
    pendingOfflineEvents(): Promise<OfflineReviewEvent[]>
    markOfflineEventsSynced(ids: string[]): Promise<void>

The frontend may compute practice previews from the offline snapshot, but permanent scoring updates belong to the server during sync. If a small local score projection is needed for immediate offline counts, keep it inside `offline.ts` and document that it is a temporary UI projection, not the durable authority.
