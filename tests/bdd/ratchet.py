from __future__ import annotations

import json
import os
from pathlib import Path
from typing import AbstractSet


def repo_root() -> Path:
    return Path(__file__).resolve().parents[2]


def implemented_commands_path(root: Path | None = None) -> Path:
    root = root or repo_root()
    return root / "specs" / "implemented-commands.json"


def load_implemented_commands(*, root: Path | None = None) -> set[str]:
    p = implemented_commands_path(root)
    if not p.exists():
        return set()
    data = json.loads(p.read_text(encoding="utf-8"))
    if not isinstance(data, list):
        raise TypeError(f"implemented-commands.json must be a JSON list, got {type(data).__name__}")
    for i, v in enumerate(data):
        if not isinstance(v, str):
            raise TypeError(f"implemented-commands.json entries must be strings (index={i}, type={type(v).__name__})")
    return set(data)


def should_xfail_not_implemented(
    *,
    command: str | None,
    error_type: str | None,
    implemented: AbstractSet[str],
) -> bool:
    if error_type != "NOT_IMPLEMENTED":
        return False
    if os.environ.get("XUEZH_STRICT_BDD") == "1":
        return False
    if not command:
        return False
    return command not in implemented
