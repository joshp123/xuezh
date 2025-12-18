from __future__ import annotations

from pathlib import Path

import pytest

from xuezh.core import audio


def test_build_convert_command_wav() -> None:
    cmd = audio.build_convert_command(Path("in.wav"), Path("out.wav"), "wav")
    assert cmd[:5] == ["ffmpeg", "-y", "-hide_banner", "-loglevel", "error"]
    assert "-ac" in cmd and "1" in cmd
    assert "-ar" in cmd and "16000" in cmd
    assert "pcm_s16le" in cmd
    assert cmd[-1] == "out.wav"


def test_build_convert_command_ogg() -> None:
    cmd = audio.build_convert_command(Path("in.wav"), Path("out.ogg"), "ogg")
    assert "libopus" in cmd
    assert "-b:a" in cmd and "24k" in cmd


def test_build_convert_command_mp3() -> None:
    cmd = audio.build_convert_command(Path("in.wav"), Path("out.mp3"), "mp3")
    assert "libmp3lame" in cmd
    assert "-b:a" in cmd and "64k" in cmd


def test_build_convert_command_rejects_unknown_format() -> None:
    with pytest.raises(ValueError):
        audio.build_convert_command(Path("in.wav"), Path("out.flac"), "flac")


def test_build_tts_command() -> None:
    cmd = audio.build_tts_command("你好", "XiaoxiaoNeural", Path("out.mp3"))
    assert cmd[:2] == ["edge-tts", "--text"]
    assert "--voice" in cmd
    assert "--write-media" in cmd


def test_mime_for_format() -> None:
    assert audio.mime_for_format("wav") == "audio/wav"
    assert audio.mime_for_format("ogg") == "audio/ogg"
    assert audio.mime_for_format("mp3") == "audio/mpeg"
    with pytest.raises(ValueError):
        audio.mime_for_format("flac")
