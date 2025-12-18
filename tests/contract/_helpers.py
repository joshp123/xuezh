from __future__ import annotations

import json
import re
import shlex
from pathlib import Path
from typing import Dict, List


def repo_root() -> Path:
    return Path(__file__).resolve().parents[2]


def load_contract() -> dict:
    p = repo_root() / "specs" / "cli" / "contract.json"
    return json.loads(p.read_text(encoding="utf-8"))


def extract_bdd_commands() -> List[str]:
    """Return raw CLI strings used in BDD feature files (quoted)."""
    root = repo_root()
    feature_dir = root / "specs" / "bdd"
    features = list(feature_dir.glob("*.feature"))
    assert features, "No .feature files found under specs/bdd"

    pattern = re.compile(r'"(xuezh [^"]+)"')
    cmds: List[str] = []
    for fp in features:
        text = fp.read_text(encoding="utf-8")
        for m in pattern.finditer(text):
            cmds.append(m.group(1))
    return cmds


def cli_to_command_id(cli: str) -> str:
    """Mechanical parse of a `xuezh ...` invocation into a command id.

    This is a structural mapping (not heuristics): it mirrors the Typer subcommand tree.
    """
    parts = shlex.split(cli)
    assert parts and parts[0] == "xuezh", f"Expected CLI starting with 'xuezh': {cli}"

    # Global commands: `xuezh <cmd>`
    if len(parts) >= 2 and parts[1] in {"version", "snapshot", "doctor", "gc"}:
        return parts[1]

    # Group commands: `xuezh <group> <verb>`
    if len(parts) < 3:
        raise AssertionError(f"Unparseable CLI (too short): {cli}")

    group = parts[1]
    verb = parts[2]

    if group in {"db", "dataset", "review", "srs", "report", "audio", "event"}:
        return f"{group}.{verb}"

    # Nested group: `xuezh content cache <verb>`
    if group == "content":
        if len(parts) < 4:
            raise AssertionError(f"Unparseable content CLI (too short): {cli}")
        subgroup = parts[2]
        verb2 = parts[3]
        return f"{group}.{subgroup}.{verb2}"

    raise AssertionError(f"Unknown CLI group in BDD: {cli}")


def parse_ticket_frontmatter(ticket_path: Path) -> Dict[str, object]:
    """Parse simple frontmatter (YAML-like) without pulling in PyYAML.

    Values for list fields are stored as JSON lists in this repo.
    """
    text = ticket_path.read_text(encoding="utf-8")
    if not text.startswith("---\n"):
        raise AssertionError(f"Ticket missing frontmatter: {ticket_path}")
    end = text.find("\n---\n", 4)
    if end == -1:
        raise AssertionError(f"Ticket frontmatter not terminated: {ticket_path}")
    fm = text[4:end].splitlines()

    out: Dict[str, object] = {}
    for line in fm:
        if not line.strip():
            continue
        if ":" not in line:
            continue
        k, v = line.split(":", 1)
        k = k.strip()
        v = v.strip()
        if v.startswith("["):
            out[k] = json.loads(v)
        elif v in {"true", "false"}:
            out[k] = (v == "true")
        else:
            # strip quotes if present
            out[k] = v.strip('"')
    return out
