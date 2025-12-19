from __future__ import annotations


def test_resolve_audio_backend_precedence(monkeypatch):
    from xuezh.cli import _resolve_audio_backend

    monkeypatch.delenv("XUEZH_AUDIO_BACKEND", raising=False)
    monkeypatch.delenv("XUEZH_AUDIO_PROCESS_VOICE_BACKEND", raising=False)

    assert (
        _resolve_audio_backend(
            cli_value=None,
            default="azure.speech",
            env_key="XUEZH_AUDIO_PROCESS_VOICE_BACKEND",
        )
        == "azure.speech"
    )

    monkeypatch.setenv("XUEZH_AUDIO_BACKEND", "global")
    assert (
        _resolve_audio_backend(
            cli_value=None,
            default="azure.speech",
            env_key="XUEZH_AUDIO_PROCESS_VOICE_BACKEND",
        )
        == "global"
    )

    monkeypatch.setenv("XUEZH_AUDIO_PROCESS_VOICE_BACKEND", "percmd")
    assert (
        _resolve_audio_backend(
            cli_value=None,
            default="azure.speech",
            env_key="XUEZH_AUDIO_PROCESS_VOICE_BACKEND",
        )
        == "percmd"
    )

    assert (
        _resolve_audio_backend(
            cli_value="cli",
            default="azure.speech",
            env_key="XUEZH_AUDIO_PROCESS_VOICE_BACKEND",
        )
        == "cli"
    )
