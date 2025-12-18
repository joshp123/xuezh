# Events and exposures (authoritative)

The engine must record **primary-source events** so the model can reason about:
- recency bias
- what the learner actually saw/did
- modality splits (reading/listening/speaking/typing)
- exposure context (story, chat, review, etc.)

This is critical for i+1 and for accurate audits.

## Event model (facts only)

An event is an append-only record:

Required fields:
- `event_id` (see `specs/id-scheme.md`)
- `ts` (ISO 8601, UTC)
- `event_type` (enum)
- `items` (list of item IDs, possibly empty)
- `modality` (enum)
- `context` (optional string, e.g., content_id or session_id)
- `payload` (JSON object, bounded)

### Event types (v1)
- `exposure` — learner saw/heard an item in context (story/listening/chat)
- `review` — review happened (even if SRS also records it)
- `pronunciation_attempt` — speaking attempt stored
- `content_served` — a cached/generated content artifact was served

## CLI commands

- `xuezh event log --type exposure --modality reading --items w_xxx,w_yyy --context ct_... --json`
- `xuezh event list --since 7d --limit 200 --json`

Notes:
- `--items` is a comma-separated list of item IDs (bounded).
- For large item lists (e.g., a story with many words), use `--items-file` pointing to a newline-delimited file of IDs.

## Snapshot integration

`xuezh snapshot` should include:
- a bounded list of recent events (for last 7 days, limit N)
- aggregated exposure counts by modality (window = snapshot window)

## ZFC constraint

Events are facts. No heuristic scoring is allowed in event logging.
