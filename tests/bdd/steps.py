from __future__ import annotations

import json
import shlex
import subprocess
import sys
from pathlib import Path
from typing import Any, Dict

import pytest
import jsonschema
from pytest_bdd import given, when, then, parsers

from tests.bdd.ratchet import load_implemented_commands, should_xfail_not_implemented


@pytest.fixture()
def workspace(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> Path:
    # Force engine to use temp workspace
    monkeypatch.setenv("XUEZH_WORKSPACE_DIR", str(tmp_path))
    return tmp_path


def _run_cli(cmd: str, workspace: Path) -> Dict[str, Any]:
    # cmd is a full string like: xuezh version --json
    # Allow placeholders used in BDD specs.
    cmd = cmd.replace('{workspace}', str(workspace))
    parts = shlex.split(cmd)
    assert parts and parts[0] == "xuezh", "BDD commands must start with 'xuezh'"
    # Execute via python -m to avoid relying on script installation
    p = subprocess.run(
        [sys.executable, "-m", "xuezh.cli", *parts[1:]],
        capture_output=True,
        text=True,
    )
    try:
        out = json.loads(p.stdout)
    except Exception as e:  # pragma: no cover
        raise AssertionError(f"CLI did not return JSON. Output:\n{p.stdout}") from e
    if out.get("ok") is True:
        assert p.returncode == 0, f"Expected exit 0 for ok envelope, got {p.returncode}\nSTDERR: {p.stderr}"
    else:
        assert p.returncode != 0, f"Expected non-zero exit for err envelope, got {p.returncode}\nSTDERR: {p.stderr}"
    return out


def _load_schema(name: str) -> Dict[str, Any]:
    repo_root = Path(__file__).resolve().parents[2]
    schema_path = repo_root / "schemas" / name
    assert schema_path.exists(), f"Missing schema file: {schema_path}"
    return json.loads(schema_path.read_text(encoding="utf-8"))


def _validate_envelope(out: Dict[str, Any]) -> None:
    schema = _load_schema("envelope.ok.schema.json" if out.get("ok") is True else "envelope.err.schema.json")
    jsonschema.validate(out, schema)


def _validate_command_schema(out: Dict[str, Any]) -> None:
    # If ok, validate against command-specific schema when present.
    if out.get("ok") is not True:
        return
    cmd_id = out.get("command")
    if not cmd_id:
        raise AssertionError("Missing command id in output")
    # Naming convention: schemas/<command>.schema.json (dots preserved)
    schema_name = f"{cmd_id}.schema.json"
    repo_root = Path(__file__).resolve().parents[2]
    schema_path = repo_root / "schemas" / schema_name
    if schema_path.exists():
        jsonschema.validate(out, json.loads(schema_path.read_text(encoding="utf-8")))


@given("a clean workspace")
def given_clean_workspace(workspace: Path) -> None:
    # workspace fixture already sets env var
    assert workspace.exists()


@given(parsers.parse('env "{key}" is "{value}"'))
def given_env(key: str, value: str, monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv(key, value)


@when(parsers.parse('the client runs "{cmd}"'), target_fixture="when_client_runs")
def when_client_runs(cmd: str, workspace: Path) -> Dict[str, Any]:
    out = _run_cli(cmd, workspace)
    _validate_envelope(out)
    return out


@then("the engine returns a valid JSON envelope")
def then_valid_envelope(when_client_runs: Dict[str, Any]) -> None:
    # envelope already validated in runner
    pass


@then("the engine returns an OK envelope")
def then_ok_or_xfail(when_client_runs: Dict[str, Any]) -> None:
    if when_client_runs.get("ok") is not True:
        err = when_client_runs.get("error", {})
        implemented = load_implemented_commands()
        if should_xfail_not_implemented(
            command=when_client_runs.get("command"),
            error_type=err.get("type"),
            implemented=implemented,
        ):
            pytest.xfail("Command not implemented yet")
        raise AssertionError(f"Expected ok=true, got error: {err}")


@then("the output matches the command-specific JSON schema")
def then_schema_or_xfail(when_client_runs: Dict[str, Any]) -> None:
    if when_client_runs.get("ok") is not True:
        err = when_client_runs.get("error", {})
        implemented = load_implemented_commands()
        if should_xfail_not_implemented(
            command=when_client_runs.get("command"),
            error_type=err.get("type"),
            implemented=implemented,
        ):
            pytest.xfail("Command not implemented yet")
        raise AssertionError(f"Expected ok=true, got error: {err}")
    _validate_command_schema(when_client_runs)


@then("the engine does not return recommendation fields")
def then_no_reco_fields(when_client_runs: Dict[str, Any]) -> None:
    forbidden = {"recommended_next", "priority_score", "should_learn", "next_best_action"}

    def walk(x):
        if isinstance(x, dict):
            for k, v in x.items():
                assert k not in forbidden, f"Forbidden key present: {k}"
                walk(v)
        elif isinstance(x, list):
            for it in x:
                walk(it)

    walk(when_client_runs)
