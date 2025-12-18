# Time + windows (policy)

This repo treats time windows as **rolling durations**, not calendar boundaries.

## Windows

Window strings are parsed as:
- `Nd` → last `N * 24h`
- `Nh` → last `N * 1h`

Example:
- `30d` means “now_utc - 30*24h … now_utc”.

## Timestamps

All stored timestamps are:
- ISO-8601 strings
- UTC

## Test determinism

Tests may freeze “now” by setting:
- `XUEZH_TEST_NOW_ISO` (ISO-8601 UTC timestamp)
