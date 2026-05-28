# Implementation spec: xuezh remote learner service for OpenClaw

- Parent RFC: `docs/rfc/2026-05-28-xuezh-client-server-protobuf.md`
- Protocol spec: `docs/rfc/2026-05-28-xuezh-client-server-protobuf-protocol.md`

## 1) Baseline Evidence That Motivated This

xuezh repo:

- `docs/cli-contract.md`: CLI JSON is the LLM/tool contract.
- `specs/cli/contract.json`: 29 command IDs requiring classification.
- `skills/chinese-learning-orchestrator/SKILL.md`: OpenClaw workflows use
  learner state, review, reports, TTS, pronunciation, content cache, and events.
- `internal/xuezh/cli/cli.go`: argv parsing and use-case logic were mixed.
- `internal/xuezh/webserver/webserver.go`: PWA routes call cram/domain packages
  directly.
- `internal/xuezh/audio/audio.go`: `ProcessVoice` calls
  `storePronunciationAttempt`, so audio is not ephemeral.
- `internal/xuezh/paths/paths.go`, `internal/xuezh/audio/azure.go`, and CLI
  backend selection accepted xuezh-specific env config.
- `flake.nix`: the OpenClaw plugin packaged xuezh, speech tools, Azure env
  requirements, and `.config/xuezh` state together.

Nix/OpenClaw evidence:

- `/Users/josh/code/nix/nixos-config/hosts/nixos/mini-server-1/xuezh.nix`
  previously ran `xuezh web serve --port 8765` with
  `XUEZH_WORKSPACE_DIR=/var/lib/xuezh`.
- `/Users/josh/code/nix/nixos-config/hosts/nixos/mini-server-1/media-ingress/Caddyfile`:
  `chinese.jjpcodes.com` proxies to `127.0.0.1:8765` with LAN-or-tailnet
  access today.
- `/Users/josh/code/nix/nixos-config/hosts/darwin/mac-mini/openclaw.nix`
  previously injected Azure secrets plus
  `XUEZH_WORKSPACE_DIR=/Users/josh/Library/Application Support/xuezh/cram-local`.
- `/Users/josh/code/nix/nixos-config/hosts/darwin/mac-mini/xuezh.nix` and
  `/Users/josh/code/nix/nixos-config/hosts/darwin/scripts/mac-mini-xuezh-server.sh`:
  stale mutable-checkout xuezh launchd wiring existed; `mac-mini.nix` did not
  import it.

## 2) Preserve and Remove

Preserve:
- Existing CLI JSON envelopes.
- Existing web/PWA HTTP endpoints.
- `audio process-voice` as one user-facing command.
- Single-user SQLite workspace semantics.
- ZFC boundary.

Remove:
- Mac OpenClaw xuezh workspace.
- Mac OpenClaw xuezh Azure/audio secrets.
- xuezh OpenClaw plugin `requiredEnv` and `stateDirs`.
- Mac mutable-checkout xuezh launchd wiring.
- xuezh-specific env vars as config inputs.
- DB mutation from low-level `audio`.
- CLI/web/audio ownership of use-case transactions.
- Invisible client/server architecture in the repo tree.

## 3) Target Product Shape

```text
api
`-- xuezh
    `-- v1
        |-- xuezh.proto
        |-- xuezh.pb.go
        `-- xuezhv1connect
cmd
`-- xuezh-go
internal/xuezh
|-- audio
|-- cli
|-- content
|-- cram
|-- events
|-- reports
|-- rpc
|-- service
`-- webserver
```

`tree -L 3 api internal/xuezh cmd` must show this shape. Codegen config lives at
repo root as `buf.yaml` and `buf.gen.yaml`; those files are build metadata, not
product concepts.

Directory contract:

- `api/xuezh/v1`: protobuf boundary and generated ConnectRPC code.
- `internal/xuezh/service`: use cases, transactions, and state ownership.
- `internal/xuezh/rpc`: protobuf transport adapter.
- `internal/xuezh/cli`: LLM-facing JSON adapter.
- `internal/xuezh/webserver`: PWA/browser HTTP adapter.
- `internal/xuezh/audio`: mechanical speech backend operations.
- `internal/xuezh/cram`, `content`, `events`, `reports`: durable learner facts
  and factual read models.

Do not add broad buckets such as `common`, `shared`, `utils`, `handlers`,
`manager`, or top-level `client`/`server`. New directories must make the product
easier to infer from `tree`.

## 4) Service Boundary

`service.App` is a concrete struct:

```go
type App struct {
    // workspace, db handle/factory, clock, audio backends, limits
}
```

Rules:

- Do not create one interface containing every use case.
- Service request/result types are plain Go structs.
- Service owns transactions and durable mutations.
- Service decides artifact writes and DB facts.
- Service receives config at construction.
- Service does not read process env directly.
- Domain packages do not import protobuf.
- Adapters define narrow interfaces only for their own tests; otherwise they call
  concrete `service.App`.

Method homes:

- `learner.go`: `Snapshot`, `LearnerState`
- `review.go`: `StartReview`, `GradeReview`, `BuryReview`, `PreviewSRS`
- `speech.go`: `SynthesizeSpeech`, `ProcessVoice`
- `content.go`: `PutContent`, `GetContent`
- `events.go`: `LogEvent`, `ListEvents`
- `reports.go`: `ReportHSK`, `ReportMastery`, `ReportDue`
- `doctor.go`: `Doctor`

## 5) Adapter Rules

`cli`:

- Parses argv and emits JSON envelopes.
- Selects local mode or client-backed mode from config.
- Rejects unsupported client-backed commands before workspace resolution.
- Converts local file inputs to bytes or IDs before RPC.
- Owns local delivery file writes for returned audio bytes.
- Does not own use-case transactions.

`webserver`:

- Registers existing PWA routes.
- Registers the ConnectRPC handler on the same `127.0.0.1:8765` process.
- Calls `service.App` for migrated use cases.
- Does not duplicate migrated business logic.

`rpc`:

- Translates protobuf to/from service structs.
- Enforces request size limits.
- Maps service errors to ConnectRPC errors.
- Does not import SQLite or domain persistence directly.

`audio`:

- Converts audio, runs TTS, runs STT/assessment, and writes requested files.
- Does not insert DB facts.
- Does not know local vs remote CLI mode.

## 6) Protobuf Codegen

- Add `api/xuezh/v1/xuezh.proto`.
- Add root `buf.yaml` and `buf.gen.yaml`.
- Generate Go protobuf and ConnectRPC code under `api/xuezh/v1/`.
- Use `buf`, `protoc-gen-go`, and `protoc-gen-connect-go` from `devenv.nix`;
  do not add bespoke generator scripts.
- Add Go deps for protobuf runtime and ConnectRPC.
- Check in generated files and verify `buf generate` is clean.

## 7) Config Changes

Add config fields:

```toml
[client]
server_url = "https://chinese.jjpcodes.com"

[workspace]
dir = "/var/lib/xuezh"
```

Rules:

- `[client].server_url` selects client-backed mode.
- `[workspace].dir` selects local/server workspace.
- A config containing both `[client]` and `[workspace]` is invalid.
- Managed hosts load `/etc/xuezh/config.toml`.
- User config remains `~/.config/xuezh/config.toml` for unmanaged developer
  workspaces.
- If host config and user config both exist, return `CONFIG_CONFLICT`.
- Production Nix writes config files instead of xuezh env vars.
- Remove xuezh-specific env override support for workspace, audio backend, and
  Azure credential resolution.

Mini-server config:

```toml
[workspace]
dir = "/var/lib/xuezh"

[audio]
process_voice_backend = "azure.speech"
convert_backend = "ffmpeg"
tts_backend = "edge-tts"
inline_max_bytes = 200000

[azure.speech]
key_file = "/run/agenix/xuezh-azure-speech-key"
region = "westeurope"
```

## 8) Client-Backed CLI Matrix

| Command ID | Behavior |
| --- | --- |
| `version` | local |
| `snapshot` | RPC |
| `learner.state` | RPC |
| `review.start` | RPC |
| `review.grade` | RPC |
| `review.bury` | RPC |
| `srs.preview` | RPC |
| `report.hsk` | RPC |
| `report.mastery` | RPC |
| `report.due` | RPC |
| `audio.tts` | RPC plus local delivery file |
| `audio.process-voice` | RPC plus XDG-cache feedback file when returned |
| `content.cache.put` | RPC; CLI uploads file bytes |
| `content.cache.get` | RPC; CLI writes XDG-cache artifact when needed |
| `event.log` | RPC |
| `event.list` | RPC |
| `doctor` | RPC server checks plus local client checks |
| `db.init` | `UNSUPPORTED_CLIENT_COMMAND` |
| `dataset.import` | `UNSUPPORTED_CLIENT_COMMAND` |
| `hellochinese.import` | `UNSUPPORTED_CLIENT_COMMAND` |
| `hellochinese.audio-backfill` | `UNSUPPORTED_CLIENT_COMMAND` |
| `travel.import` | `UNSUPPORTED_CLIENT_COMMAND` |
| `pleco.score-import` | `UNSUPPORTED_CLIENT_COMMAND` |
| `cram.overview` | `UNSUPPORTED_CLIENT_COMMAND` |
| `cram.audio-backfill` | `UNSUPPORTED_CLIENT_COMMAND` |
| `cram.next` | `UNSUPPORTED_CLIENT_COMMAND` |
| `cram.grade` | `UNSUPPORTED_CLIENT_COMMAND` |
| `audio.convert` | `UNSUPPORTED_CLIENT_COMMAND` |
| `gc` | `UNSUPPORTED_CLIENT_COMMAND` |

`web serve` is a process entrypoint and remains server-local.

## 9) Audio Contract

TTS:

1. CLI parses `audio tts`.
2. CLI derives server output format from the local `--out` extension using
   current TTS extension rules.
3. CLI calls `SynthesizeSpeech`.
4. Server writes canonical artifact under `/var/lib/xuezh`.
5. Server returns artifact metadata and inline audio bytes.
6. CLI writes `--out` as a local delivery file when provided.
7. CLI envelope separates server artifact metadata from local delivery path.

Pronunciation:

1. CLI reads the input voice file into a bounded request.
2. Server writes upload/normalized/transcript/assessment/feedback artifacts.
3. `service.App.ProcessVoice` inserts one pronunciation attempt.
4. Server returns assessment, transcript, attempt ID, artifacts, and feedback
   audio bytes when generated and under cap.
5. CLI envelope keeps existing actionable assessment/transcript shape.

Use protocol caps. Oversize input returns a typed error before backend work.

## 10) Implementation Order

Slice 1: contract guard.

- Add `CONFIG_CONFLICT` and `UNSUPPORTED_CLIENT_COMMAND` to typed errors.
- Test every command ID is local, RPC, or rejected in client-backed mode.
- Test rejected client commands create no workspace.

Slice 2: config and path cleanup.

- Add `[client]` and `[workspace]` config parsing.
- Make workspace resolution read config.
- Make client-backed mode unavailable when workspace config is present.
- Remove xuezh-specific env override support; tests use temp config files.

Slice 3: service boundary.

- Add `internal/xuezh/service.App`.
- Move learner, review, reports, content, events, and speech use cases behind it.
- Change CLI and webserver to call `service.App`.
- Keep domain packages focused on mechanical/domain primitives.

Slice 4: audio mutation extraction.

- Split audio pipeline result from pronunciation-attempt insertion.
- Move pronunciation-attempt insertion into `service.App.ProcessVoice`.
- Keep low-level audio package DB-free.

Slice 5: protobuf API.

- Add `api/xuezh/v1/xuezh.proto`.
- Generate Go and ConnectRPC code.
- Add `internal/xuezh/rpc` server and client adapters.
- Register RPC handler in `xuezh web serve`.

Slice 6: client-backed CLI.

- Route supported command IDs through RPC.
- Preserve JSON envelopes.
- Materialize local delivery audio only for audio commands.
- Reject unsupported commands with `UNSUPPORTED_CLIENT_COMMAND`.

Slice 7: Nix/OpenClaw cutover.

- Move xuezh Azure key config and audio backend dependencies to mini-server
  runtime.
- Write mini-server `/etc/xuezh/config.toml`; remove
  `environment.XUEZH_WORKSPACE_DIR` from `xuezh.service`.
- Put `ffmpeg` and `edge-tts` on the `xuezh.service` runtime path.
- Reduce xuezh `openclawPlugin.packages` to the xuezh CLI.
- Remove Mac OpenClaw xuezh workspace env and Azure secret declarations.
- Remove xuezh plugin `requiredEnv` and `stateDirs`.
- Add Mac `/etc/xuezh/config.toml` with `[client].server_url`.
- Delete the stale Mac mutable-checkout xuezh launchd module and script.
- Split Caddy so RPC paths import `tailnet_access`, not `lan_or_tailnet_access`.

Slice 8: docs/skills.

- Update `docs/README.md`, `docs/cli-contract.md`, and
  `specs/audio-backends.md` for client-backed mode and TOML-only config.
- Update `skills/chinese-learning-orchestrator/SKILL.md` and duplicate
  `skills/xuezh/SKILL.md`: teach client-backed CLI, not protobuf; remove
  Mac-local workspace/env guidance; use temp delivery paths for `audio tts
  --out`; omit OpenClaw TTS backend selection; treat server artifact paths as
  audit metadata.
- Update nix smoke tests that currently inject Mac-local xuezh env.

## 11) Tests

Unit:

- config local mode, client mode, invalid mixed mode
- `CONFIG_CONFLICT` and `UNSUPPORTED_CLIENT_COMMAND` are known error types
- host/user config conflict returns `CONFIG_CONFLICT`
- path resolution does not create workspace in client-backed rejected commands
- `audio.ProcessVoice` performs no DB insert
- `service.App.ProcessVoice` inserts one pronunciation row
- review grade commits score rows, review events, and session state together
- `service.App` methods return unchanged local CLI data for migrated commands

RPC:

- in-process `GetLearnerState`
- in-process `GradeReview`
- in-process `SynthesizeSpeech`
- in-process `ProcessVoice`
- request size cap returns invalid-argument
- TTS text and content upload caps reject before backend work
- protobuf responses do not contain CLI envelopes
- generated protobuf code is clean after `buf generate`

CLI:

- local envelopes unchanged for migrated commands
- client-backed `learner state` matches RPC result
- client-backed `audio tts --out ...` writes local delivery file
- client-backed RPC failure returns `ok:false`
- client-backed unsupported command returns `UNSUPPORTED_CLIENT_COMMAND`
- every `specs/cli/contract.json` command ID is classified

Infra:

- mini-server `xuezh.service` active
- existing `/api/learner/state` healthy
- ConnectRPC reachable from Mac over tailnet
- ConnectRPC rejected outside tailnet
- Mac launchd/plugin env and age secrets have no xuezh workspace or Azure material
- xuezh OpenClaw plugin has no `requiredEnv` and no xuezh state dirs
- active Mac host config does not import mutable-checkout xuezh launchd wiring
- `tree -L 3 api internal/xuezh cmd` shows the target ontology

## 12) Rollback

Before Mac cutover:

- Disable `[client].server_url` or revert code.
- No data movement required.

After Mac cutover:

- Fix mini-server service/Caddy first.
- Do not copy `/var/lib/xuezh` back to Mac.
