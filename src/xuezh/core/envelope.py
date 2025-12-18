from __future__ import annotations

from dataclasses import dataclass, asdict
from typing import Any, Dict, List, Optional


@dataclass
class Artifact:
    path: str
    mime: str
    purpose: str
    bytes: Optional[int] = None


def ok(
    *,
    command: str,
    data: Dict[str, Any] | None = None,
    artifacts: List[Artifact] | None = None,
    truncated: bool = False,
    limits: Dict[str, Any] | None = None,
    schema_version: str = "1",
) -> Dict[str, Any]:
    return {
        "ok": True,
        "schema_version": schema_version,
        "command": command,
        "data": data or {},
        "artifacts": [asdict(a) for a in (artifacts or [])],
        "truncated": truncated,
        "limits": limits or {},
    }


def err(
    *,
    command: str,
    error_type: str,
    message: str,
    details: Dict[str, Any] | None = None,
    schema_version: str = "1",
) -> Dict[str, Any]:
    return {
        "ok": False,
        "schema_version": schema_version,
        "command": command,
        "error": {
            "type": error_type,
            "message": message,
            "details": details or {},
        },
    }
