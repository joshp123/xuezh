from __future__ import annotations

from dataclasses import dataclass, asdict
from datetime import datetime
import json
import shutil
from pathlib import Path
from uuid import uuid4

from xuezh.core import clock, jsonio, paths
from xuezh.core.envelope import Artifact
from xuezh.core.process import ensure_tool, run_checked

SUPPORTED_FORMATS = {"wav", "ogg", "mp3"}
VOICE_ALIASES = {
    "XiaoxiaoNeural": "zh-CN-XiaoxiaoNeural",
}


@dataclass(frozen=True)
class AudioResult:
    data: dict
    artifacts: list[Artifact]


def mime_for_format(fmt: str) -> str:
    fmt = fmt.lower()
    if fmt == "wav":
        return "audio/wav"
    if fmt == "ogg":
        return "audio/ogg"
    if fmt == "mp3":
        return "audio/mpeg"
    raise ValueError(f"Unsupported audio format: {fmt}")


@dataclass(frozen=True)
class SttResult:
    data: dict
    artifacts: list[Artifact]
    truncated: bool
    limits: dict


def build_convert_command(in_path: Path, out_path: Path, fmt: str) -> list[str]:
    fmt = fmt.lower()
    if fmt not in SUPPORTED_FORMATS:
        raise ValueError(f"Unsupported audio format: {fmt}")

    cmd = [
        "ffmpeg",
        "-y",
        "-hide_banner",
        "-loglevel",
        "error",
        "-i",
        str(in_path),
    ]

    if fmt == "wav":
        cmd += ["-ac", "1", "-ar", "16000", "-c:a", "pcm_s16le"]
    elif fmt == "ogg":
        cmd += ["-ac", "1", "-ar", "48000", "-c:a", "libopus", "-b:a", "24k"]
    elif fmt == "mp3":
        cmd += ["-ac", "1", "-ar", "44100", "-c:a", "libmp3lame", "-b:a", "64k"]

    cmd.append(str(out_path))
    return cmd


def build_tts_command(text: str, voice: str, out_path: Path) -> list[str]:
    return [
        "edge-tts",
        "--text",
        text,
        "--voice",
        voice,
        "--write-media",
        str(out_path),
    ]


def build_stt_command(in_path: Path, out_dir: Path) -> list[str]:
    return [
        "whisper",
        str(in_path),
        "--model",
        "tiny",
        "--output_format",
        "json",
        "--output_dir",
        str(out_dir),
        "--language",
        "zh",
        "--task",
        "transcribe",
    ]


def _artifact_for(path: Path, fmt: str, purpose: str) -> Artifact:
    workspace = paths.ensure_workspace()
    rel_path = str(path.relative_to(workspace))
    return Artifact(path=rel_path, mime=mime_for_format(fmt), purpose=purpose, bytes=path.stat().st_size)


def _artifact_path(prefix: str, ext: str, now: datetime) -> Path:
    root = paths.ensure_workspace()
    day_path = root / "artifacts" / now.strftime("%Y") / now.strftime("%m") / now.strftime("%d")
    day_path.mkdir(parents=True, exist_ok=True)
    filename = f"{prefix}-{now.strftime('%Y%m%dT%H%M%SZ')}.{ext}"
    return day_path / filename


def _extract_transcript(raw: dict) -> dict:
    transcript = {
        "text": (raw.get("text") or "").strip(),
        "segments": [
            {
                "start": segment.get("start"),
                "end": segment.get("end"),
                "text": (segment.get("text") or "").strip(),
            }
            for segment in raw.get("segments", []) or []
        ],
    }
    if raw.get("language"):
        transcript["language"] = raw.get("language")
    return transcript


def convert_audio(*, in_path: str, out_path: str, fmt: str, backend: str) -> AudioResult:
    if backend != "ffmpeg":
        raise ValueError(f"Unsupported backend: {backend}")

    input_path = Path(in_path).expanduser()
    if not input_path.exists():
        raise FileNotFoundError(f"Input file not found: {input_path}")

    output_path = paths.resolve_in_workspace(out_path)
    output_path.parent.mkdir(parents=True, exist_ok=True)

    ensure_tool("ffmpeg")
    cmd = build_convert_command(input_path, output_path, fmt)
    run_checked(cmd)

    artifact = _artifact_for(output_path, fmt, "converted_audio")
    data = {
        "in": str(input_path),
        "out": artifact.path,
        "format": fmt,
        "backend": {"id": backend, "features": ["convert"]},
    }
    return AudioResult(data=data, artifacts=[artifact])


def tts_audio(*, text: str, voice: str, out_path: str, backend: str) -> AudioResult:
    if backend != "edge-tts":
        raise ValueError(f"Unsupported backend: {backend}")

    resolved_voice = VOICE_ALIASES.get(voice, voice)
    output_path = paths.resolve_in_workspace(out_path)
    output_path.parent.mkdir(parents=True, exist_ok=True)

    ensure_tool("edge-tts")
    temp_path = output_path.parent / f".tts-{uuid4().hex}.mp3"
    cmd = build_tts_command(text, resolved_voice, temp_path)
    run_checked(cmd)

    fmt = output_path.suffix.lstrip(".").lower() or "ogg"
    if fmt not in SUPPORTED_FORMATS:
        fmt = "ogg"
    ensure_tool("ffmpeg")
    try:
        cmd = build_convert_command(temp_path, output_path, fmt)
        run_checked(cmd)
    finally:
        if temp_path.exists():
            temp_path.unlink()

    artifact = _artifact_for(output_path, fmt, "tts_audio")
    data = {
        "text": text,
        "voice": resolved_voice,
        "out": artifact.path,
        "backend": {"id": backend, "features": ["tts"]},
    }
    return AudioResult(data=data, artifacts=[artifact])


def stt_audio(*, in_path: str, backend: str, max_bytes: int = 200_000) -> SttResult:
    if backend != "whisper":
        raise ValueError(f"Unsupported backend: {backend}")

    input_path = Path(in_path).expanduser()
    if not input_path.exists():
        raise FileNotFoundError(f"Input file not found: {input_path}")

    ensure_tool("whisper")
    now = clock.now_utc()
    temp_dir = paths.ensure_workspace() / "artifacts" / f".stt-{uuid4().hex}"
    temp_dir.mkdir(parents=True, exist_ok=True)

    try:
        cmd = build_stt_command(input_path, temp_dir)
        run_checked(cmd)

        output_json = temp_dir / input_path.with_suffix(".json").name
        if not output_json.exists():
            raise FileNotFoundError(f"Whisper output not found: {output_json}")

        raw = json.loads(output_json.read_text(encoding="utf-8"))
        transcript = _extract_transcript(raw)

        transcript_path = _artifact_path(f"stt-{input_path.stem}", "json", now)
        transcript_path.write_text(jsonio.dumps(transcript), encoding="utf-8")
    finally:
        shutil.rmtree(temp_dir, ignore_errors=True)

    artifact = Artifact(
        path=str(transcript_path.relative_to(paths.ensure_workspace())),
        mime="application/json",
        purpose="transcript",
        bytes=transcript_path.stat().st_size,
    )

    data = {
        "in": str(input_path),
        "transcript": transcript,
        "backend": {"id": backend, "features": ["stt"]},
    }
    limits = {"max_bytes": max_bytes}
    envelope = {
        "ok": True,
        "schema_version": "1",
        "command": "audio.stt",
        "data": data,
        "artifacts": [asdict(artifact)],
        "truncated": False,
        "limits": limits,
    }
    payload = jsonio.dumps(envelope)
    if len(payload.encode("utf-8")) > max_bytes:
        data = {
            "in": str(input_path),
            "transcript": {"spill_artifact": artifact.path},
            "backend": {"id": backend, "features": ["stt"]},
        }
        return SttResult(data=data, artifacts=[artifact], truncated=True, limits=limits)

    return SttResult(data=data, artifacts=[artifact], truncated=False, limits=limits)
