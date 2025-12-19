# pytest configuration hooks.
#
# Policy: No skipped tests. Skips hide real problems and break the "idiot-proof" spec discipline.
# If something cannot run in this environment, use xfail with a clear reason (and fix it later).

from __future__ import annotations

import os
from pathlib import Path
import pytest

# Register pytest-bdd step definitions as a pytest plugin so fixtures are discoverable.
pytest_plugins = ["tests.bdd.steps"]

_SKIP_COUNT = 0


def pytest_configure() -> None:
    if "XUEZH_CONFIG_PATH" not in os.environ:
        path = Path(__file__).resolve().parents[1] / ".xuezh-test-config.toml"
        os.environ["XUEZH_CONFIG_PATH"] = str(path)


def pytest_runtest_logreport(report: pytest.TestReport) -> None:
    global _SKIP_COUNT
    if report.when == "setup" and report.outcome == "skipped":
        _SKIP_COUNT += 1


def pytest_sessionfinish(session: pytest.Session, exitstatus: int) -> None:
    if _SKIP_COUNT > 0:
        pytest.exit(f"Skipped tests are not allowed (skipped={_SKIP_COUNT}).", returncode=2)
