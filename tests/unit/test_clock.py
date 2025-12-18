from __future__ import annotations

from datetime import timezone, timedelta

import pytest

from xuezh.core.clock import now_utc, parse_duration, parse_utc_iso


def test_parse_utc_iso_accepts_z_and_returns_utc() -> None:
    dt = parse_utc_iso("2025-12-18T00:00:00Z")
    assert dt.tzinfo == timezone.utc
    assert dt.isoformat() == "2025-12-18T00:00:00+00:00"


def test_now_utc_uses_test_override(monkeypatch) -> None:
    monkeypatch.setenv("XUEZH_TEST_NOW_ISO", "2025-12-18T01:02:03+00:00")
    assert now_utc().isoformat() == "2025-12-18T01:02:03+00:00"


def test_parse_duration_days_and_hours() -> None:
    assert parse_duration("30d") == timedelta(days=30)
    assert parse_duration("24h") == timedelta(hours=24)


def test_parse_duration_rejects_invalid() -> None:
    with pytest.raises(ValueError):
        parse_duration("30")
