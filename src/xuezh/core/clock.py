from __future__ import annotations

import os
import re
from datetime import datetime, timedelta, timezone

_DURATION_RE = re.compile(r"^(?P<n>[0-9]+)(?P<unit>[dh])$")


def now_utc() -> datetime:
    """Return the current time in UTC.

    Tests can override via `XUEZH_TEST_NOW_ISO` to make time-based logic deterministic.
    """
    override = os.environ.get("XUEZH_TEST_NOW_ISO")
    if override:
        return parse_utc_iso(override)
    return datetime.now(timezone.utc)


def parse_utc_iso(s: str) -> datetime:
    """Parse an ISO-8601 timestamp into a UTC datetime.

    Accepts either an explicit offset (e.g. `...+00:00`) or `Z`.
    """
    if s.endswith("Z"):
        s = f"{s[:-1]}+00:00"
    dt = datetime.fromisoformat(s)
    if dt.tzinfo is None:
        raise ValueError("Timestamp must be timezone-aware (include +00:00 or Z)")
    return dt.astimezone(timezone.utc)


def parse_duration(s: str) -> timedelta:
    """Parse `Nd` or `Nh` into a timedelta."""
    m = _DURATION_RE.match(s)
    if not m:
        raise ValueError(f"Invalid duration: {s!r} (expected Nd or Nh, e.g. 30d, 24h)")
    n = int(m.group("n"))
    unit = m.group("unit")
    if unit == "d":
        return timedelta(days=n)
    return timedelta(hours=n)
