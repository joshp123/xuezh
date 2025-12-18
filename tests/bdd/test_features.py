"""Executable BDD harness.

These tests intentionally *xfail* while the CLI commands return NOT_IMPLEMENTED.
Once a ticket implements a command, the scenario becomes strict automatically.

This keeps the repo usable for RGR (tests green early), while still making the BDD
suite executable and continuously checked.
"""

from __future__ import annotations

from pytest_bdd import scenarios

# Load all feature files from specs/bdd
scenarios("../../specs/bdd")
