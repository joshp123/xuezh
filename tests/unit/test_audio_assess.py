from __future__ import annotations

from xuezh.core.audio import assess_from_transcript


def test_assess_from_transcript_exact_match() -> None:
    out = assess_from_transcript("你好", "你好")
    assert out["exact_match"] is True


def test_assess_from_transcript_ignores_whitespace() -> None:
    out = assess_from_transcript("你 好", "你好")
    assert out["exact_match"] is True


def test_assess_from_transcript_mismatch() -> None:
    out = assess_from_transcript("你好", "您好")
    assert out["exact_match"] is False
