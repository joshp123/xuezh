from __future__ import annotations

from dataclasses import dataclass, asdict
from datetime import datetime
import json
import shutil
from pathlib import Path
from uuid import uuid4

from xuezh.core import clock, db, jsonio, paths
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


@dataclass(frozen=True)
class AssessResult:
    data: dict
    artifacts: list[Artifact]


@dataclass(frozen=True)
class ProcessVoiceResult:
    data: dict
    artifacts: list[Artifact]


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


def convert_audio(
    *,
    in_path: str,
    out_path: str,
    fmt: str,
    backend: str,
    purpose: str = "converted_audio",
) -> AudioResult:
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

    artifact = _artifact_for(output_path, fmt, purpose)
    data = {
        "in": str(input_path),
        "out": artifact.path,
        "format": fmt,
        "backend": {"id": backend, "features": ["convert"]},
    }
    return AudioResult(data=data, artifacts=[artifact])


def tts_audio(
    *,
    text: str,
    voice: str,
    out_path: str,
    backend: str,
    purpose: str = "tts_audio",
) -> AudioResult:
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

    artifact = _artifact_for(output_path, fmt, purpose)
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


def _normalize_text(text: str) -> str:
    return "".join(text.strip().split()).lower()


def assess_from_transcript(ref_text: str, transcript_text: str) -> dict:
    ref_norm = _normalize_text(ref_text)
    transcript_norm = _normalize_text(transcript_text)
    return {
        "ref_text": ref_text,
        "transcript_text": transcript_text,
        "exact_match": ref_norm == transcript_norm,
        "note": "local_v0_placeholder",
    }


def assess_audio(*, ref_text: str, in_path: str, backend: str) -> AssessResult:
    if backend != "local":
        raise ValueError(f"Unsupported backend: {backend}")

    stt_result = stt_audio(in_path=in_path, backend="whisper")
    transcript_text = stt_result.data.get("transcript", {}).get("text", "")
    assessment = assess_from_transcript(ref_text, transcript_text)

    now = clock.now_utc()
    assessment_path = _artifact_path("assessment", "json", now)
    assessment_path.write_text(jsonio.dumps(assessment), encoding="utf-8")
    assessment_artifact = Artifact(
        path=str(assessment_path.relative_to(paths.ensure_workspace())),
        mime="application/json",
        purpose="assessment",
        bytes=assessment_path.stat().st_size,
    )

    data = {
        "ref_text": ref_text,
        "in": str(Path(in_path).expanduser()),
        "assessment": assessment,
        "backend": {"id": backend, "features": ["assessment"]},
    }
    artifacts = [assessment_artifact, *stt_result.artifacts]
    return AssessResult(data=data, artifacts=artifacts)


def _store_pronunciation_attempt(
    *,
    backend_id: str,
    artifacts: list[Artifact],
    summary: dict,
) -> str:
    import sqlite3
    import ulid

    attempt_id = str(ulid.new())
    db_path = db.init_db()
    conn = sqlite3.connect(db_path)
    try:
        conn.execute(
            """
            INSERT INTO pronunciation_attempts (id, item_id, ts, backend_id, artifacts_json, summary_json)
            VALUES (?, ?, ?, ?, ?, ?)
            """,
            (
                attempt_id,
                None,
                clock.now_utc().isoformat(),
                backend_id,
                json.dumps([asdict(a) for a in artifacts], ensure_ascii=False, sort_keys=True),
                json.dumps(summary, ensure_ascii=False, sort_keys=True),
            ),
        )
        conn.commit()
    finally:
        conn.close()
    return attempt_id


def process_voice(*, in_path: str, ref_text: str, backend: str) -> ProcessVoiceResult:
    if backend != "local":
        raise ValueError(f"Unsupported backend: {backend}")

    normalized = convert_audio(
        in_path=in_path,
        out_path=str(_artifact_path("normalized-input", "wav", clock.now_utc())),
        fmt="wav",
        backend="ffmpeg",
        purpose="normalized_input",
    )
    normalized_path = paths.resolve_in_workspace(normalized.artifacts[0].path)
    stt_result = stt_audio(in_path=str(normalized_path), backend="whisper")
    transcript_text = stt_result.data.get("transcript", {}).get("text", "")
    assessment = assess_from_transcript(ref_text, transcript_text)

    assessment_path = _artifact_path("assessment", "json", clock.now_utc())
    assessment_path.write_text(jsonio.dumps(assessment), encoding="utf-8")
    assessment_artifact = Artifact(
        path=str(assessment_path.relative_to(paths.ensure_workspace())),
        mime="application/json",
        purpose="assessment",
        bytes=assessment_path.stat().st_size,
    )

    feedback = tts_audio(
        text=ref_text,
        voice="XiaoxiaoNeural",
        out_path=str(_artifact_path("feedback-voice", "ogg", clock.now_utc())),
        backend="edge-tts",
        purpose="feedback_voice_note",
    )

    artifacts = [
        *normalized.artifacts,
        *stt_result.artifacts,
        assessment_artifact,
        *feedback.artifacts,
    ]
    artifacts_index = {artifact.purpose: artifact.path for artifact in artifacts}
    summary = {"assessment": assessment, "artifacts_index": artifacts_index}
    attempt_id = _store_pronunciation_attempt(
        backend_id=backend,
        artifacts=artifacts,
        summary=summary,
    )

    data = {
        "ref_text": ref_text,
        "backend": {"id": backend, "features": ["assessment", "tts", "stt", "convert"]},
        "artifacts_index": artifacts_index,
        "attempt_id": attempt_id,
    }
    return ProcessVoiceResult(data=data, artifacts=artifacts)
