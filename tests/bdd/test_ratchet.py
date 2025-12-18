from __future__ import annotations

from tests.bdd.ratchet import should_xfail_not_implemented


def test_ratchet_xfails_only_when_command_not_marked_implemented(monkeypatch) -> None:
    monkeypatch.delenv("XUEZH_STRICT_BDD", raising=False)

    assert (
        should_xfail_not_implemented(
            command="snapshot",
            error_type="NOT_IMPLEMENTED",
            implemented=set(),
        )
        is True
    )

    assert (
        should_xfail_not_implemented(
            command="snapshot",
            error_type="NOT_IMPLEMENTED",
            implemented={"snapshot"},
        )
        is False
    )


def test_ratchet_respects_strict_bdd(monkeypatch) -> None:
    monkeypatch.setenv("XUEZH_STRICT_BDD", "1")
    assert (
        should_xfail_not_implemented(
            command="snapshot",
            error_type="NOT_IMPLEMENTED",
            implemented=set(),
        )
        is False
    )
