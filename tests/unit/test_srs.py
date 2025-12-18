from datetime import datetime, timezone

from xuezh.core import srs


def test_schedule_next_due_override():
    now = datetime(2025, 1, 2, tzinfo=timezone.utc)
    due_at, rule = srs.schedule_next_due(grade=3, now=now, rule="sm2", next_due="2025-01-05T00:00:00+00:00")
    assert due_at == "2025-01-05T00:00:00+00:00"
    assert rule is None


def test_schedule_next_due_rule_default():
    now = datetime(2025, 1, 2, tzinfo=timezone.utc)
    due_at, rule = srs.schedule_next_due(grade=4, now=now, rule=None, next_due=None)
    assert rule == "sm2"
    assert due_at == "2025-01-09T00:00:00+00:00"
