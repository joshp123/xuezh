from __future__ import annotations

import json


def test_inline_payload_truncates_words(monkeypatch):
    from xuezh.core import audio

    monkeypatch.setattr(audio, "INLINE_DETAIL_MAX_BYTES", 200)
    assessment = {
        "overall": {"accuracy_score": 1.0},
        "words": [{"word": "x" * 50} for _ in range(20)],
    }
    transcript = {
        "text": "hi",
        "words": [{"word": "x" * 50} for _ in range(20)],
    }
    artifacts_index = {"assessment": "artifacts/assessment.json", "transcript": "artifacts/transcript.json"}

    assessment_inline, transcript_inline, truncated = audio._inline_pronunciation_payload(
        assessment=assessment,
        transcript=transcript,
        artifacts_index=artifacts_index,
    )

    assert truncated is True
    assert "words" not in assessment_inline
    assert "words" not in transcript_inline


def test_inline_payload_minimal_spill(monkeypatch):
    from xuezh.core import audio

    monkeypatch.setattr(audio, "INLINE_DETAIL_MAX_BYTES", 150)
    transcript_text = "x" * 1000
    assessment = {"overall": {"accuracy_score": 1.0}}
    transcript = {"text": transcript_text}
    artifacts_index = {"assessment": "artifacts/assessment.json", "transcript": "artifacts/transcript.json"}

    assessment_inline, transcript_inline, truncated = audio._inline_pronunciation_payload(
        assessment=assessment,
        transcript=transcript,
        artifacts_index=artifacts_index,
    )

    assert truncated is True
    assert assessment_inline.get("spill_artifact") == "artifacts/assessment.json"
    assert transcript_inline.get("spill_artifact") == "artifacts/transcript.json"
    assert "text_preview" not in transcript_inline


def test_assess_audio_local_spill(monkeypatch, tmp_path):
    from xuezh.core import audio
    from xuezh.core.envelope import Artifact

    monkeypatch.setenv("XUEZH_WORKSPACE_DIR", str(tmp_path))
    spill_path = tmp_path / "artifacts" / "spill.json"
    spill_path.parent.mkdir(parents=True, exist_ok=True)
    spill_path.write_text(json.dumps({"text": "你好"}), encoding="utf-8")
    artifact = Artifact(
        path=str(spill_path.relative_to(tmp_path)),
        mime="application/json",
        purpose="transcript",
        bytes=spill_path.stat().st_size,
    )

    def _fake_stt_audio(*, in_path: str, backend: str, max_bytes: int = 200_000):
        return audio.SttResult(
            data={"transcript": {"spill_artifact": artifact.path}},
            artifacts=[artifact],
            truncated=True,
            limits={"max_bytes": max_bytes},
        )

    monkeypatch.setattr(audio, "stt_audio", _fake_stt_audio)

    result = audio.assess_audio(ref_text="你好", in_path="noop.wav", backend="local")
    assert result.data["assessment"]["transcript_text"] == "你好"
