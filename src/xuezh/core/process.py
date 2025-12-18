from __future__ import annotations

from dataclasses import dataclass
import shutil
import subprocess
from typing import Sequence


class ToolMissingError(RuntimeError):
    def __init__(self, tool: str) -> None:
        super().__init__(f"Required tool not found on PATH: {tool}")
        self.tool = tool


class ProcessFailedError(RuntimeError):
    def __init__(self, cmd: Sequence[str], returncode: int, stdout: str, stderr: str) -> None:
        super().__init__(f"Process failed with code {returncode}: {' '.join(cmd)}")
        self.cmd = list(cmd)
        self.returncode = returncode
        self.stdout = stdout
        self.stderr = stderr


@dataclass(frozen=True)
class ProcessResult:
    stdout: str
    stderr: str
    returncode: int


def ensure_tool(name: str) -> str:
    path = shutil.which(name)
    if not path:
        raise ToolMissingError(name)
    return path


def run_checked(cmd: Sequence[str]) -> ProcessResult:
    proc = subprocess.run(list(cmd), capture_output=True, text=True)
    if proc.returncode != 0:
        raise ProcessFailedError(cmd, proc.returncode, proc.stdout, proc.stderr)
    return ProcessResult(stdout=proc.stdout, stderr=proc.stderr, returncode=proc.returncode)
