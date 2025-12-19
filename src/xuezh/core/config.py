from __future__ import annotations

import os
from pathlib import Path
import tomllib

_CONFIG_CACHE: dict | None = None


def config_path() -> Path:
    override = os.environ.get("XUEZH_CONFIG_PATH")
    if override:
        return Path(override).expanduser()
    base = os.environ.get("XDG_CONFIG_HOME")
    root = Path(base).expanduser() if base else (Path.home() / ".config")
    return root / "xuezh" / "config.toml"


def load_config() -> dict:
    path = config_path()
    if not path.exists():
        return {}
    try:
        data = tomllib.loads(path.read_text(encoding="utf-8"))
    except (OSError, tomllib.TOMLDecodeError) as exc:
        raise ValueError(f"Invalid config file: {path}") from exc
    if not isinstance(data, dict):
        raise ValueError(f"Invalid config file structure: {path}")
    return data


def get_config() -> dict:
    global _CONFIG_CACHE
    if _CONFIG_CACHE is None:
        _CONFIG_CACHE = load_config()
    return _CONFIG_CACHE


def reset_config_cache() -> None:
    global _CONFIG_CACHE
    _CONFIG_CACHE = None


def get_config_value(*keys: str, default: object | None = None) -> object | None:
    current: object = get_config()
    for key in keys:
        if not isinstance(current, dict) or key not in current:
            return default
        current = current[key]
    return current
