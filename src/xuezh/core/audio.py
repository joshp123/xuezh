from __future__ import annotations

from dataclasses import dataclass, asdict, field
from datetime import datetime
import json
import os
import shutil
from pathlib import Path
from uuid import uuid4

from xuezh.core import clock, config as config_core, db, jsonio, paths
from xuezh.core.envelope import Artifact
from xuezh.core.process import ToolMissingError, ensure_tool, run_checked

SUPPORTED_FORMATS = {"wav", "ogg", "mp3"}
VOICE_ALIASES = {
    "XiaoxiaoNeural": "zh-CN-XiaoxiaoNeural",
}
INLINE_DETAIL_MAX_BYTES_DEFAULT = 200_000


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
    truncated: bool = False
    limits: dict = field(default_factory=dict)


class AzureSpeechError(RuntimeError):
    def __init__(self, kind: str, message: str, details: dict | None = None) -> None:
        super().__init__(message)
        self.kind = kind
        self.details = details or {}


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


def _summarize_detail(payload: dict) -> dict:
    summary = dict(payload)
    summary.pop("words", None)
    summary.pop("segments", None)
    return summary


def _payload_bytes(*, assessment: dict, transcript: dict) -> int:
    payload = {"assessment": assessment, "transcript": transcript}
    return len(jsonio.dumps(payload).encode("utf-8"))


def _inline_detail_max_bytes() -> int:
    config_value = config_core.get_config_value("audio", "inline_max_bytes")
    if isinstance(config_value, int) and config_value > 0:
        return config_value
    env_value = os.environ.get("XUEZH_AUDIO_INLINE_MAX_BYTES")
    if env_value:
        try:
            return int(env_value)
        except ValueError:
            pass
    return INLINE_DETAIL_MAX_BYTES_DEFAULT


def _minimal_assessment(assessment: dict, artifacts_index: dict) -> dict:
    minimal: dict = {}
    overall = assessment.get("overall")
    if isinstance(overall, dict) and overall:
        minimal["overall"] = overall
    if "exact_match" in assessment:
        minimal["exact_match"] = assessment.get("exact_match")
    if "note" in assessment:
        minimal["note"] = assessment.get("note")
    assessment_artifact = artifacts_index.get("assessment")
    if assessment_artifact:
        minimal["spill_artifact"] = assessment_artifact
    return minimal


def _minimal_transcript(transcript: dict, artifacts_index: dict, preview_len: int) -> dict:
    minimal: dict = {}
    text = transcript.get("text")
    if isinstance(text, str) and preview_len > 0:
        minimal["text_preview"] = text[:preview_len]
        minimal["text_truncated"] = len(text) > preview_len
    transcript_artifact = artifacts_index.get("transcript")
    if transcript_artifact:
        minimal["spill_artifact"] = transcript_artifact
    return minimal


def _inline_pronunciation_payload(
    *,
    assessment: dict,
    transcript: dict,
    artifacts_index: dict,
) -> tuple[dict, dict, bool]:
    max_bytes = _inline_detail_max_bytes()
    detail_bytes = _payload_bytes(assessment=assessment, transcript=transcript)
    if detail_bytes <= max_bytes:
        return assessment, transcript, False
    assessment_summary = _summarize_detail(assessment)
    transcript_summary = _summarize_detail(transcript)
    summary_bytes = _payload_bytes(assessment=assessment_summary, transcript=transcript_summary)
    if summary_bytes <= max_bytes:
        return assessment_summary, transcript_summary, True
    preview_len = 2000
    text = transcript.get("text")
    if isinstance(text, str):
        preview_len = min(len(text), preview_len)
    else:
        preview_len = 0
    assessment_min = _minimal_assessment(assessment, artifacts_index)
    transcript_min = _minimal_transcript(transcript, artifacts_index, preview_len)
    minimal_bytes = _payload_bytes(assessment=assessment_min, transcript=transcript_min)
    if minimal_bytes <= max_bytes:
        return assessment_min, transcript_min, True
    transcript_min = _minimal_transcript(transcript, artifacts_index, 0)
    return assessment_min, transcript_min, True


def assess_from_transcript(ref_text: str, transcript_text: str) -> dict:
    ref_norm = _normalize_text(ref_text)
    transcript_norm = _normalize_text(transcript_text)
    return {
        "ref_text": ref_text,
        "transcript_text": transcript_text,
        "exact_match": ref_norm == transcript_norm,
        "note": "local_v0_placeholder",
    }


def _write_json_artifact(payload: dict, *, purpose: str, prefix: str, now: datetime) -> Artifact:
    path = _artifact_path(prefix, "json", now)
    path.write_text(jsonio.dumps(payload), encoding="utf-8")
    return Artifact(
        path=str(path.relative_to(paths.ensure_workspace())),
        mime="application/json",
        purpose=purpose,
        bytes=path.stat().st_size,
    )


def _load_speechsdk():
    try:
        import azure.cognitiveservices.speech as speechsdk  # type: ignore
    except Exception as exc:  # pragma: no cover - import guard
        raise ToolMissingError("azure-cognitiveservices-speech") from exc
    return speechsdk


def _azure_pronunciation_assess(*, ref_text: str, wav_path: Path) -> tuple[dict, dict, dict]:
    speechsdk = _load_speechsdk()

    config_key = None
    config_region = None
    config_section = config_core.get_config_value("azure", "speech")
    if isinstance(config_section, dict):
        config_key = config_section.get("key")
        key_file = config_section.get("key_file")
        if key_file:
            try:
                config_key = Path(str(key_file)).expanduser().read_text(encoding="utf-8").strip()
            except OSError:
                config_key = None
        config_region = config_section.get("region")

    key = config_key or os.environ.get("AZURE_SPEECH_KEY")
    region = config_region or os.environ.get("AZURE_SPEECH_REGION")
    if not key or not region:
        missing = []
        if not key:
            missing.append("AZURE_SPEECH_KEY or config.azure.speech.key/key_file")
        if not region:
            missing.append("AZURE_SPEECH_REGION or config.azure.speech.region")
        raise AzureSpeechError(
            "auth",
            f"Azure Speech credentials missing ({', '.join(missing)})",
            {"missing": missing},
        )

    speech_config = speechsdk.SpeechConfig(subscription=key, region=region)
    speech_config.speech_recognition_language = "zh-CN"

    audio_config = speechsdk.audio.AudioConfig(filename=str(wav_path))
    recognizer = speechsdk.SpeechRecognizer(speech_config=speech_config, audio_config=audio_config)

    pa_config = speechsdk.PronunciationAssessmentConfig(
        reference_text=ref_text,
        grading_system=speechsdk.PronunciationAssessmentGradingSystem.HundredMark,
        granularity=speechsdk.PronunciationAssessmentGranularity.Phoneme,
        enable_miscue=True,
    )
    pa_config.apply_to(recognizer)

    result = recognizer.recognize_once_async().get()
    if result.reason == speechsdk.ResultReason.Canceled:
        details = speechsdk.CancellationDetails.from_result(result)
        error_text = details.error_details or ""
        lowered = error_text.lower()
        if "quota" in lowered or "limit" in lowered or "429" in lowered:
            raise AzureSpeechError("quota", "Azure Speech quota exceeded", {"error_details": error_text})
        if "401" in lowered or "403" in lowered or "unauthorized" in lowered:
            raise AzureSpeechError("auth", "Azure Speech authentication failed", {"error_details": error_text})
        raise AzureSpeechError("backend", "Azure Speech request failed", {"error_details": error_text})

    raw_json_text = result.properties.get(speechsdk.PropertyId.SpeechServiceResponse_JsonResult) or "{}"
    raw_json = json.loads(raw_json_text)

    pa_result = speechsdk.PronunciationAssessmentResult(result)
    overall = {
        "accuracy_score": getattr(pa_result, "accuracy_score", None),
        "fluency_score": getattr(pa_result, "fluency_score", None),
        "completeness_score": getattr(pa_result, "completeness_score", None),
        "pronunciation_score": getattr(pa_result, "pronunciation_score", None),
    }

    nbest = (raw_json.get("NBest") or [{}])[0]
    display_text = nbest.get("Display") or nbest.get("DisplayText") or raw_json.get("DisplayText") or raw_json.get("Text")
    words = []
    for word in nbest.get("Words", []) or []:
        words.append(
            {
                "word": word.get("Word"),
                "accuracy_score": word.get("PronunciationAssessment", {}).get("AccuracyScore"),
                "error_type": word.get("PronunciationAssessment", {}).get("ErrorType"),
                "syllables": word.get("Syllables"),
                "phonemes": word.get("Phonemes"),
            }
        )

    transcript = {
        "text": display_text or "",
    }

    assessment = {
        "reference_text": ref_text,
        "transcript_text": transcript["text"],
        "overall": overall,
        "words": words,
    }
    return assessment, transcript, raw_json


def assess_audio(*, ref_text: str, in_path: str, backend: str) -> AssessResult:
    if backend == "local":
        stt_result = stt_audio(in_path=in_path, backend="whisper")
        transcript = stt_result.data.get("transcript", {}) or {}
        spill_path = transcript.get("spill_artifact")
        if spill_path:
            try:
                transcript_path = paths.resolve_in_workspace(spill_path)
                transcript = json.loads(transcript_path.read_text(encoding="utf-8"))
            except (OSError, json.JSONDecodeError):
                transcript = {"spill_artifact": spill_path}
        transcript_text = transcript.get("text", "")
        assessment = assess_from_transcript(ref_text, transcript_text)

        now = clock.now_utc()
        assessment_artifact = _write_json_artifact(assessment, purpose="assessment", prefix="assessment", now=now)

        data = {
            "ref_text": ref_text,
            "in": str(Path(in_path).expanduser()),
            "assessment": assessment,
            "backend": {"id": backend, "features": ["assessment"]},
        }
        artifacts = [assessment_artifact, *stt_result.artifacts]
        return AssessResult(data=data, artifacts=artifacts)

    if backend == "azure.speech":
        normalized = convert_audio(
            in_path=in_path,
            out_path=str(_artifact_path("normalized-input", "wav", clock.now_utc())),
            fmt="wav",
            backend="ffmpeg",
            purpose="normalized_input",
        )
        normalized_path = paths.resolve_in_workspace(normalized.artifacts[0].path)
        assessment, transcript, raw_json = _azure_pronunciation_assess(ref_text=ref_text, wav_path=normalized_path)

        now = clock.now_utc()
        assessment_artifact = _write_json_artifact(assessment, purpose="assessment", prefix="assessment", now=now)
        transcript_artifact = _write_json_artifact(transcript, purpose="transcript", prefix="transcript", now=now)
        raw_artifact = _write_json_artifact(raw_json, purpose="azure_response", prefix="azure-response", now=now)

        data = {
            "ref_text": ref_text,
            "in": str(Path(in_path).expanduser()),
            "assessment": assessment,
            "backend": {"id": backend, "features": ["assessment"]},
        }
        artifacts = [*normalized.artifacts, transcript_artifact, assessment_artifact, raw_artifact]
        return AssessResult(data=data, artifacts=artifacts)

    raise ValueError(f"Unsupported backend: {backend}")


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


def process_voice(*, in_path: str, ref_text: str, backend: str = "azure.speech") -> ProcessVoiceResult:
    if backend not in {"local", "azure.speech"}:
        raise ValueError(f"Unsupported backend: {backend}")

    normalized = convert_audio(
        in_path=in_path,
        out_path=str(_artifact_path("normalized-input", "wav", clock.now_utc())),
        fmt="wav",
        backend="ffmpeg",
        purpose="normalized_input",
    )
    normalized_path = paths.resolve_in_workspace(normalized.artifacts[0].path)

    if backend == "local":
        stt_result = stt_audio(in_path=str(normalized_path), backend="whisper")
        transcript = stt_result.data.get("transcript", {}) or {}
        spill_path = transcript.get("spill_artifact")
        if spill_path:
            try:
                transcript_path = paths.resolve_in_workspace(spill_path)
                transcript = json.loads(transcript_path.read_text(encoding="utf-8"))
            except (OSError, json.JSONDecodeError):
                transcript = {"spill_artifact": spill_path}
        transcript_text = transcript.get("text", "")
        assessment = assess_from_transcript(ref_text, transcript_text)
        assessment_artifact = _write_json_artifact(
            assessment, purpose="assessment", prefix="assessment", now=clock.now_utc()
        )
        transcript_artifacts = stt_result.artifacts
    else:
        assessment, transcript, raw_json = _azure_pronunciation_assess(ref_text=ref_text, wav_path=normalized_path)
        assessment_artifact = _write_json_artifact(
            assessment, purpose="assessment", prefix="assessment", now=clock.now_utc()
        )
        transcript_artifact = _write_json_artifact(
            transcript, purpose="transcript", prefix="transcript", now=clock.now_utc()
        )
        raw_artifact = _write_json_artifact(
            raw_json, purpose="azure_response", prefix="azure-response", now=clock.now_utc()
        )
        transcript_artifacts = [transcript_artifact, raw_artifact]

    feedback = tts_audio(
        text=ref_text,
        voice="XiaoxiaoNeural",
        out_path=str(_artifact_path("feedback-voice", "ogg", clock.now_utc())),
        backend="edge-tts",
        purpose="feedback_voice_note",
    )

    artifacts = [
        *normalized.artifacts,
        *transcript_artifacts,
        assessment_artifact,
        *feedback.artifacts,
    ]
    artifacts_index = {artifact.purpose: artifact.path for artifact in artifacts}
    assessment_inline, transcript_inline, inline_truncated = _inline_pronunciation_payload(
        assessment=assessment,
        transcript=transcript,
        artifacts_index=artifacts_index,
    )
    summary = {"assessment": assessment, "artifacts_index": artifacts_index}
    try:
        _store_pronunciation_attempt(
            backend_id=backend,
            artifacts=artifacts,
            summary=summary,
        )
    except Exception:
        pass

    if backend == "local":
        features = ["assessment", "tts", "stt", "convert"]
    else:
        features = ["assessment", "tts", "convert", "azure.speech"]

    data = {
        "ref_text": ref_text,
        "backend": {"id": backend, "features": features},
        "artifacts_index": artifacts_index,
        "assessment": assessment_inline,
        "transcript": transcript_inline,
    }
    limits = {}
    if inline_truncated:
        limits = {"inline_bytes_max": _inline_detail_max_bytes()}
    return ProcessVoiceResult(data=data, artifacts=artifacts, truncated=inline_truncated, limits=limits)
