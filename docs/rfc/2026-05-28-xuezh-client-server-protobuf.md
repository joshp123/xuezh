# RFC: xuezh remote learner service for OpenClaw

- Date: 2026-05-28
- Status: Draft
- Audience: Josh, xuezh implementers, OpenClaw/Nix operators
- Implementation spec: `docs/rfc/2026-05-28-xuezh-client-server-protobuf-implementation.md`
- Protocol spec: `docs/rfc/2026-05-28-xuezh-client-server-protobuf-protocol.md`

## 1) Decision

Make `mini-server-1` the only durable xuezh state owner and make Mac OpenClaw
call it through a typed protobuf API. Keep the xuezh CLI as the LLM-facing tool:
on the server it runs local use cases; on the Mac it becomes a thin client for
the learning workflows OpenClaw actually uses.

No DB copy, no rsync, no SSH wrapper, no second Mac workspace, no generic
`Command(args)` RPC.

Use ConnectRPC because xuezh already serves Go HTTP behind Caddy, and generated
protobuf routes give a typed host-to-host boundary without a second daemon.

Mac-local xuezh DBs are not imported during cutover. The server DB is the source
of truth.

## 2) Problem

The live xuezh service is on `mini-server-1`:

```text
xuezh.service
/var/lib/xuezh
https://chinese.jjpcodes.com
```

Mac OpenClaw is still configured with Mac-local xuezh state:

```text
/Users/josh/Library/Application Support/xuezh/cram-local
```

That is split brain. It can read stale learner facts and write new facts to the
wrong SQLite DB. This includes `audio process-voice`: today it calls
`storePronunciationAttempt`, so local pronunciation assessment mutates learner
history.

## 3) Product Workflows

The OpenClaw skill uses xuezh for these workflows:

- Load learner context: `learner.state`
- Review and mutate SRS facts: `review.start`, `review.grade`, `review.bury`
- Produce progress facts: `snapshot`, `report.hsk`, `report.mastery`,
  `report.due`, `srs.preview`
- Produce and assess speech: `audio.tts`, `audio.process-voice`
- Store generated learning material and exposure facts: `content.cache.put`,
  `content.cache.get`, `event.log`, `event.list`
- Diagnose the wiring: `doctor`

The RFC is scoped to those workflows. Web/PWA cram endpoints keep their existing
HTTP surface and call the same internal service. Legacy `cram.*` CLI commands
are not part of the remote OpenClaw API.

## 4) Non-Negotiables

- `mini-server-1` owns durable xuezh state.
- Mac OpenClaw must not set `XUEZH_WORKSPACE_DIR` or `XUEZH_DB_PATH`.
- Mac OpenClaw must not hold xuezh Azure Speech secrets.
- Mac OpenClaw must not run stateful xuezh commands against a local DB.
- Client-backed CLI must not silently fall back to local execution.
- Protobuf must not wrap the existing CLI as a generic remote shell.
- CLI JSON envelopes stay the LLM contract.
- Protobuf is the host-to-host contract.
- xuezh stays single-user.
- xuezh stays ZFC-compliant: no recommendations, lesson planning, ranking, or
  "what next" policy.
- The product ontology must be visible from `tree -L 3 api internal/xuezh cmd`.

## 5) Runtime Topology

```text
Mac OpenClaw agent
  -> xuezh CLI
  -> ConnectRPC/protobuf over HTTPS
  -> Caddy on chinese.jjpcodes.com
  -> xuezh web/RPC process on 127.0.0.1:8765
  -> service.App
  -> SQLite, artifacts, audio backends in /var/lib/xuezh
```

There is still one xuezh daemon on the server. Do not add a second RPC daemon or
sidecar. The existing `xuezh web serve` process registers both the PWA HTTP
routes and the ConnectRPC handler.

## 6) CLI Modes

The CLI has two modes, selected by config file, not flags or xuezh env vars.

Local mode:

- No `[client]` config.
- Used by the server service and by developer workspaces.
- Runs `service.App` in process.
- Opens or creates the configured workspace as commands require.

Client-backed mode:

- `[client].server_url` is set.
- Used by Mac OpenClaw.
- Supported learning commands call RPC.
- Rejected commands return a typed JSON error before workspace resolution.
- The client writes temporary delivery media only for OpenClaw handoff, never a
  xuezh DB or canonical artifact store.

## 7) Config

Target production config is TOML, not xuezh-specific env vars.

- Mac OpenClaw config contains only `[client].server_url`.
- Mini-server config contains `[workspace]`, `[audio]`, and `[azure.speech]`.
- The implementation spec owns the exact TOML.

Rules:

- xuezh-specific env vars are not config inputs after this migration.
- Production OpenClaw and `xuezh.service` use config files.
- Managed hosts use `/etc/xuezh/config.toml`.
- User config remains `~/.config/xuezh/config.toml` only for unmanaged
  developer workspaces.
- If host config and user config both exist, xuezh fails with `CONFIG_CONFLICT`
  instead of choosing one silently.
- Tests use temporary config files, not xuezh env overrides.

## 8) Client Command Contract

The implementation spec owns the exact command matrix. The rule is:

- OpenClaw learning workflows call RPC.
- `version` stays local.
- operator, import, web/PWA cram, bulk audio, `audio.convert`, `gc`, and
  `web serve` commands are server-local and rejected from client-backed mode.

Rejected commands return `UNSUPPORTED_CLIENT_COMMAND` with `server_url` in error
details. They must not create `~/.clawdbot`, `~/Library/Application
Support/xuezh`, or any other xuezh workspace.

For file-bearing commands, the client consumes local paths before RPC:

- `audio.process-voice --in`: upload bytes, not a Mac path
- `audio.tts --out`: write returned bytes to local delivery path
- `content.cache.put --in`: upload bytes, not a Mac path
- `event.log --items-file`: parse item IDs locally, then send IDs
- `content.cache.get`: write returned bytes to XDG cache when a local artifact
  path is needed for the JSON envelope

Response translation is typed. The Mac CLI receives generated protobuf structs,
maps them field-by-field into the existing command JSON envelope, then writes
JSON to stdout for OpenClaw. It does not parse remote JSON. `ReportPayload` and
`LearnerState` carry bounded dynamic values already owned by the CLI contract;
audio/content commands also materialize returned inline bytes into local
delivery files before emitting the envelope. ConnectRPC errors map to existing
`ok:false` typed error envelopes.

## 9) Audio Semantics

Audio has two kinds of files.

Server artifacts:

- Written by `service.App` under `/var/lib/xuezh`.
- Recorded in DB/artifact metadata by server use cases.
- Used for audit, retention, and learner history.
- Written before DB rows reference them; failures can leave unreferenced files
  for GC, but not DB rows pointing at missing artifacts.

Client delivery files:

- Written by the Mac CLI only for OpenClaw delivery.
- Live under caller-provided output paths or an XDG cache temp path.
- Are not xuezh artifacts, not DB state, and not synced back.

`audio.tts` in client-backed mode:

- CLI calls `SynthesizeSpeech`.
- Server writes canonical TTS artifact.
- Server returns inline audio bytes plus server artifact metadata.
- Server rejects text or generated audio that exceeds v1 RPC caps.
- Because current `audio.tts` requires `--out`, client-backed mode treats
  `--out` as a local delivery path, not as a xuezh workspace artifact path.
- The RPC output format uses current local TTS extension rules.
- JSON includes the server artifact and the local delivery path separately.

`audio.process-voice` in client-backed mode:

- CLI reads bounded local input bytes.
- Server normalizes audio, runs assessment/STT, writes artifacts, and records one
  pronunciation attempt.
- Server returns feedback audio bytes when generated and under the response cap.
- CLI writes returned feedback audio under XDG cache and reports that path as
  delivery scratch.

V1 does not stream or chunk. The protocol spec owns request/response caps.

## 10) State Ownership

Server-owned:

- SQLite DB
- cram/review state
- SRS score rows
- review events
- pronunciation attempts
- generated content records
- exposure/event log
- canonical artifacts
- retention/GC policy

Review mutations are ACID at the SQLite boundary: score rows, review events,
and review/session state commit together or not at all.

Mac-owned:

- inbound voice file before upload
- returned audio bytes after response
- returned cached content bytes after response
- local delivery files needed by OpenClaw

Mac-owned files have no learning semantics.

Server artifact paths in RPC responses are server workspace-relative metadata.
The Mac client treats them as opaque audit references, not local filesystem
paths to open.

## 11) Code Ontology

The tree is part of the design. A maintainer running
`tree -L 3 api internal/xuezh cmd` must see the product shape before opening a
file: protobuf edge, application service, adapters, speech, review/cram,
content, events, and factual reports.

Target ontology:

```text
api/xuezh/v1/              # protobuf API and generated Go
internal/xuezh/service/    # application use cases and transactions
internal/xuezh/rpc/        # ConnectRPC server/client adapters
internal/xuezh/cli/        # JSON CLI adapter
internal/xuezh/webserver/  # PWA/browser HTTP adapter
internal/xuezh/audio/      # mechanical audio backend calls
internal/xuezh/cram/       # cram/review domain primitives
internal/xuezh/events/     # event facts
internal/xuezh/content/    # generated content cache
internal/xuezh/reports/    # bounded factual reports
```

If a concept is important enough to explain in this RFC, it gets a named home
in that tree. If it is not a product concept, keep it local to the owning
package.

`service.App` is concrete. Do not create a broad `Service` interface. Use narrow
interfaces only at adapter/test seams that need them.

Reject vague buckets such as `common`, `shared`, `utils`, `handlers`, `manager`,
or top-level `client`/`server` directories.

## 12) Security

Primary boundary: Tailscale plus Caddy route policy.

Rules:

- `chinese.jjpcodes.com` remains the host.
- Existing PWA/JSON routes remain LAN-or-tailnet.
- ConnectRPC routes must be tailnet-only.
- Tailnet access is the owner boundary for V1.
- Mutating RPC routes must not be reachable through the LAN matcher.
- No public internet RPC access.
- No app-level bearer token in V1. Caddy rejects non-tailnet RPC before xuezh.
- If RPC ever leaves tailnet-only access, add one age-managed static bearer token
  at the RPC adapter. Do not build users, OAuth, RBAC, sessions, or an API
  gateway.

Expected Caddy shape:

```text
handle /xuezh.v1.XuezhService/* {
  import tailnet_access
  reverse_proxy 127.0.0.1:8765
}
handle {
  import lan_or_tailnet_access
  reverse_proxy 127.0.0.1:8765
}
```

## 13) Acceptance Gates

Code gates:

- `./scripts/check.sh` passes.
- CLI JSON schemas still pass for unchanged commands.
- `CONFIG_CONFLICT` and `UNSUPPORTED_CLIENT_COMMAND` are registered typed errors.
- Every command ID in `specs/cli/contract.json` is classified as remote, local,
  or rejected in client-backed mode.
- `tree -L 3 api internal/xuezh cmd` shows the target ontology.
- CLI and webserver call `service.App` for migrated use cases.
- Domain/service code does not import generated protobuf packages.
- RPC code does not import SQLite directly.
- `audio.ProcessVoice` no longer writes `pronunciation_attempts`.
- `service.App.ProcessVoice` writes exactly one pronunciation row.
- Rejected client commands create no workspace.

Runtime gates:

- `mini-server-1` serves xuezh from `/var/lib/xuezh`.
- `mini-server-1` has `/etc/xuezh/config.toml` for workspace, audio, and Azure
  key config; `xuezh.service` does not set `XUEZH_WORKSPACE_DIR`.
- `https://chinese.jjpcodes.com/api/learner/state` remains healthy.
- RPC endpoint is reachable from Mac over tailnet.
- RPC endpoint is rejected outside tailnet.
- Mac OpenClaw has no xuezh workspace env.
- Mac OpenClaw has no xuezh Azure/audio secret env.
- Mac Nix config has no xuezh Azure age secrets.
- The xuezh OpenClaw plugin does not require xuezh env vars or xuezh state dirs.
- The xuezh OpenClaw skill describes client-backed CLI use and does not teach
  Mac-local workspace, env, or artifact paths.
- Mac Nix config has no xuezh launchd service for a mutable checkout.
- Mac `xuezh doctor --json` reports client-backed mode and server reachability.
- Mac `audio.process-voice` mutates only `/var/lib/xuezh`.
- Mac `audio.tts` returns sendable local audio without creating Mac xuezh state.

## 14) Rejected Designs

| Design | Rejection |
| --- | --- |
| Generic `Command(args)` RPC | keeps the CLI monolith as the architecture |
| Hand-written REST clone of the CLI | duplicates contracts without improving boundaries |
| Existing PWA JSON as OpenClaw API | cements browser routing as service logic and does not cover speech workflows |
| RPC for every CLI command | exposes operator/bulk tools OpenClaw does not need |
| SSH wrapper | couples OpenClaw to shell access and remote paths |
| DB/artifact sync | creates conflict and corruption problems |
| Local audio with remote event writes | still risks split pronunciation history |
| NFS-mounted workspace | hides the split brain instead of removing it |
| Separate RPC daemon | adds a process without adding a product boundary |
| Auth platform | wrong scope for a single-user tailnet service |

## 15) Review Standard

Reject the implementation if:

- protobuf lands before the service boundary
- protobuf messages become domain structs
- CLI keeps owning use-case logic
- webserver duplicates migrated use-case logic
- audio keeps writing learner history
- Mac config keeps a xuezh workspace
- client-backed RPC errors fall back to local state
- new directories hide the architecture from `tree`
