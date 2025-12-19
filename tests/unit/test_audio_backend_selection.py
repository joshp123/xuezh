from __future__ import annotations


def test_resolve_audio_backend_precedence(monkeypatch, tmp_path):
    from xuezh.cli import _resolve_audio_backend
    from xuezh.core import config as config_core

    monkeypatch.delenv("XUEZH_AUDIO_BACKEND", raising=False)
    monkeypatch.delenv("XUEZH_AUDIO_PROCESS_VOICE_BACKEND", raising=False)
    config_path = tmp_path / "config.toml"
    config_path.write_text(
        "[audio]\nprocess_voice_backend = \"cfg\"\nbackend_global = \"cfg_global\"\n",
        encoding="utf-8",
    )
    monkeypatch.setenv("XUEZH_CONFIG_PATH", str(config_path))
    config_core.reset_config_cache()

    assert (
        _resolve_audio_backend(
            cli_value=None,
            default="azure.speech",
            env_key="XUEZH_AUDIO_PROCESS_VOICE_BACKEND",
            config_key="process_voice_backend",
        )
        == "cfg"
    )

    monkeypatch.setenv("XUEZH_AUDIO_BACKEND", "global")
    assert (
        _resolve_audio_backend(
            cli_value=None,
            default="azure.speech",
            env_key="XUEZH_AUDIO_PROCESS_VOICE_BACKEND",
            config_key="process_voice_backend",
        )
        == "cfg"
    )

    monkeypatch.setenv("XUEZH_AUDIO_PROCESS_VOICE_BACKEND", "percmd")
    assert (
        _resolve_audio_backend(
            cli_value=None,
            default="azure.speech",
            env_key="XUEZH_AUDIO_PROCESS_VOICE_BACKEND",
            config_key="process_voice_backend",
        )
        == "cfg"
    )

    assert (
        _resolve_audio_backend(
            cli_value="cli",
            default="azure.speech",
            env_key="XUEZH_AUDIO_PROCESS_VOICE_BACKEND",
            config_key="process_voice_backend",
        )
        == "cli"
    )


def test_resolve_audio_backend_env_fallback(monkeypatch, tmp_path):
    from xuezh.cli import _resolve_audio_backend
    from xuezh.core import config as config_core

    monkeypatch.setenv("XUEZH_CONFIG_PATH", str(tmp_path / "missing.toml"))
    config_core.reset_config_cache()
    monkeypatch.setenv("XUEZH_AUDIO_BACKEND", "global")

    assert (
        _resolve_audio_backend(
            cli_value=None,
            default="azure.speech",
            env_key="XUEZH_AUDIO_PROCESS_VOICE_BACKEND",
            config_key="process_voice_backend",
        )
        == "global"
    )
