# pytest configuration hooks.
#
# Policy: No skipped tests. Skips hide real problems and break the "idiot-proof" spec discipline.
# If something cannot run in this environment, use xfail with a clear reason (and fix it later).

from __future__ import annotations

import pytest

# Import pytest-bdd step definitions
from tests.bdd import steps  # noqa: F401

_SKIP_COUNT = 0


def pytest_runtest_logreport(report: pytest.TestReport) -> None:
    global _SKIP_COUNT
    if report.when == "setup" and report.outcome == "skipped":
        _SKIP_COUNT += 1


def pytest_sessionfinish(session: pytest.Session, exitstatus: int) -> None:
    if _SKIP_COUNT > 0:
        pytest.exit(f"Skipped tests are not allowed (skipped={_SKIP_COUNT}).", returncode=2)
