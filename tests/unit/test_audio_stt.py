from __future__ import annotations

from pathlib import Path

from xuezh.core import audio


def test_build_stt_command() -> None:
    cmd = audio.build_stt_command(Path("in.wav"), Path("outdir"))
    assert cmd[0] == "whisper"
    assert "--model" in cmd
    assert "tiny" in cmd
    assert "--output_format" in cmd
    assert "json" in cmd
    assert "--output_dir" in cmd
