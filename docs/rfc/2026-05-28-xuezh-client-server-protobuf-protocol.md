# Protocol spec: xuezh remote learner service

- Parent RFC: `docs/rfc/2026-05-28-xuezh-client-server-protobuf.md`
- Target file: `api/xuezh/v1/xuezh.proto`

## 1) Rules

- Protobuf is the host-to-host contract.
- CLI JSON envelopes are not protobuf messages.
- Domain/service structs are not generated protobuf structs.
- RPC adapters translate between protobuf and service structs.
- The API exposes OpenClaw learning workflows, not every CLI command.
- Use typed messages for core workflows.
- Use `google.protobuf.Struct` only for bounded report payloads, dynamic
  third-party speech assessment/transcript details, and doctor diagnostic
  details.
- Client paths are never server artifact paths.

## 2) Service

```proto
syntax = "proto3";

package xuezh.v1;

option go_package = "github.com/joshp123/xuezh/api/xuezh/v1;xuezhv1";

import "google/protobuf/struct.proto";
import "google/protobuf/timestamp.proto";

service XuezhService {
  rpc GetLearnerState(GetLearnerStateRequest) returns (LearnerState);
  rpc GetSnapshot(GetSnapshotRequest) returns (ReportPayload);

  rpc StartReview(StartReviewRequest) returns (StartReviewResponse);
  rpc GradeReview(GradeReviewRequest) returns (GradeReviewResponse);
  rpc BuryReview(BuryReviewRequest) returns (BuryReviewResponse);
  rpc PreviewSRS(PreviewSRSRequest) returns (ReportPayload);

  rpc ReportHSK(ReportHSKRequest) returns (ReportPayload);
  rpc ReportMastery(ReportMasteryRequest) returns (ReportPayload);
  rpc ReportDue(ReportDueRequest) returns (ReportPayload);

  rpc SynthesizeSpeech(SynthesizeSpeechRequest) returns (SynthesizeSpeechResponse);
  rpc ProcessVoice(ProcessVoiceRequest) returns (ProcessVoiceResponse);

  rpc PutContent(PutContentRequest) returns (ContentRecord);
  rpc GetContent(GetContentRequest) returns (GetContentResponse);

  rpc LogEvent(LogEventRequest) returns (EventRecord);
  rpc ListEvents(ListEventsRequest) returns (ListEventsResponse);

  rpc Doctor(DoctorRequest) returns (DoctorResponse);
}
```

No `RunCommand`, `Execute`, `CommandRequest`, or CLI argv field is allowed.

## 3) Shared Messages

```proto
message ServerArtifact {
  string path = 1;
  string mime = 2;
  string purpose = 3;
  optional int64 bytes = 4;
}

message InlineFile {
  bytes data = 1;
  string mime = 2;
  string filename = 3;
}

message BackendInfo {
  string id = 1;
  repeated string features = 2;
}

message ReportPayload {
  google.protobuf.Struct data = 1;
  repeated ServerArtifact artifacts = 2;
  bool truncated = 3;
  google.protobuf.Struct limits = 4;
}
```

`ServerArtifact.path` is workspace-relative on the server. Clients can display it
for audit but must not treat it as a local path.

`ReportPayload` is only for `snapshot`, reports, and `srs.preview`, whose
existing contract is bounded JSON facts. Do not use it for command routing,
audio requests, review mutations, content writes, or event writes.
Replacing it with typed report messages requires changing the existing report
JSON contracts in the same patch.

## 4) Learner State

```proto
message GetLearnerStateRequest {}

message LearnerState {
  google.protobuf.Timestamp generated_at = 1;
  google.protobuf.Timestamp changed_at = 2;
  string state_hash = 3;
  int32 learned_score = 4;
  repeated string columns = 5;
  repeated LearnerCardRow cards = 6;
}

message LearnerCardRow {
  repeated google.protobuf.Value values = 1;
}
```

`LearnerCardRow` uses `Value` because `learner.state` is an intentional compact
columnar mixed-type contract. This is not a precedent for generic protobuf.

## 5) Review

```proto
message StartReviewRequest {
  int32 limit = 1;
}

message ReviewItem {
  string item_id = 1;
  google.protobuf.Timestamp due_at = 2;
  string review_type = 3;
}

message StartReviewResponse {
  repeated ReviewItem recall_items = 1;
  repeated ReviewItem pronunciation_items = 2;
  google.protobuf.Timestamp generated_at = 3;
}

message GradeReviewRequest {
  string item_id = 1;
  optional int32 recall = 2;
  optional int32 pronunciation = 3;
  google.protobuf.Timestamp next_due = 4;
  string rule = 5;
}

message GradeReviewResponse {
  string item_id = 1;
  optional int32 recall_grade = 2;
  google.protobuf.Timestamp recall_next_due = 3;
  optional string recall_rule_applied = 4;
  optional int32 pronunciation_grade = 5;
  google.protobuf.Timestamp pronunciation_next_due = 6;
  optional string pronunciation_rule_applied = 7;
}

message BuryReviewRequest {
  string item_id = 1;
  string reason = 2;
}

message BuryReviewResponse {
  string item_id = 1;
  string reason = 2;
  google.protobuf.Timestamp next_due = 3;
}

message PreviewSRSRequest {
  int32 days = 1;
}
```

The CLI maps legacy `review grade --grade N` to `recall = N`; the RPC contract
does not carry a duplicate `grade` field.

## 6) Reports

```proto
message GetSnapshotRequest {
  string window = 1;
  int32 due_limit = 2;
  int32 evidence_limit = 3;
  int32 max_bytes = 4;
}

message ReportHSKRequest {
  string level = 1;
  string window = 2;
  int32 max_items = 3;
  int32 max_bytes = 4;
  bool include_chars = 5;
}

message ReportMasteryRequest {
  string item_type = 1;
  string window = 2;
  int32 max_items = 3;
  int32 max_bytes = 4;
}

message ReportDueRequest {
  int32 limit = 1;
  int32 max_bytes = 2;
}
```

## 7) Audio

```proto
message SynthesizeSpeechRequest {
  string text = 1;
  string voice = 2;
  string output_format = 3;
}

message SynthesizeSpeechResponse {
  string text = 1;
  string voice = 2;
  BackendInfo backend = 3;
  repeated ServerArtifact artifacts = 4;
  InlineFile audio = 5;
}

message ProcessVoiceRequest {
  bytes audio = 1;
  string filename = 2;
  string mime = 3;
  string ref_text = 4;
}

message ProcessVoiceResponse {
  string attempt_id = 1;
  string ref_text = 2;
  BackendInfo backend = 3;
  google.protobuf.Struct assessment = 4;
  google.protobuf.Struct transcript = 5;
  repeated ServerArtifact artifacts = 6;
  InlineFile feedback_audio = 7;
  bool truncated = 8;
  google.protobuf.Struct limits = 9;
}
```

Server request cap starts at 25 MB. Oversize requests return ConnectRPC
invalid-argument. There is no streaming in v1.

`SynthesizeSpeechRequest.text` is capped at 1,000 Unicode code points.
`output_format` is derived by the CLI from the local `--out` extension using
the current TTS rule: `wav`, `ogg`, and `mp3` are accepted; missing or
unsupported extensions resolve to `ogg`. It is not a new CLI flag.
`audio.tts` has no remote `rate` field because the CLI contract has no `--rate`;
bulk cram audio rates stay server-local.
Generated audio and feedback audio are capped at 5 MB inline. Oversize outputs
return ConnectRPC resource-exhausted because the client cannot send a server
artifact path to OpenClaw.

`ProcessVoiceRequest` does not include `item_id` because the current CLI
contract does not accept one; the pronunciation attempt row keeps the existing
nullable item binding.

## 8) Content

```proto
message PutContentRequest {
  string type = 1;
  string key = 2;
  string filename = 3;
  bytes content = 4;
}

message GetContentRequest {
  string type = 1;
  string key = 2;
}

message ContentRecord {
  string type = 1;
  string key = 2;
  string content_id = 3;
  repeated ServerArtifact artifacts = 4;
}

message GetContentResponse {
  ContentRecord record = 1;
  bytes content = 2;
}
```

`PutContentRequest.filename` is the basename of the local `--in` path; the
server uses it only for the current extension rule. `PutContentRequest.content`
and `GetContentResponse.content` are capped at 1 MB in v1. Oversize content
returns ConnectRPC resource-exhausted.

## 9) Events

```proto
message LogEventRequest {
  string event_type = 1;
  string modality = 2;
  repeated string items = 3;
  optional string context = 4;
}

message EventRecord {
  string event_id = 1;
  string event_type = 2;
  google.protobuf.Timestamp ts = 3;
  string modality = 4;
  repeated string items = 5;
  optional string context = 6;
}

message ListEventsRequest {
  string since = 1;
  int32 limit = 2;
}

message ListEventsResponse {
  repeated EventRecord events = 1;
}
```

Events are facts. The API must not accept plans, recommendations, or priority
scores.

There is no generic event payload in v1. The current event contract is event
type, modality, item IDs, timestamp, and optional context.

## 10) Doctor

```proto
message DoctorRequest {}

message DoctorResponse {
  string server_version = 1;
  string workspace_role = 2;
  string workspace_path = 3;
  repeated DoctorCheck checks = 4;
}

message DoctorCheck {
  string name = 1;
  bool ok = 2;
  google.protobuf.Struct details = 3;
}
```

Doctor reports server version, server workspace role, server workspace path, and
server-side audio backend readiness. The CLI combines this with local
client-backed config checks in the JSON envelope.
