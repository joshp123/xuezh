from __future__ import annotations

import hashlib
import re

import ulid

# ID regexes (authoritative in specs/id-scheme.md)
WORD_ID_RE = re.compile(r"^w_[0-9a-f]{12}$")
GRAMMAR_ID_RE = re.compile(r"^g_[0-9a-f]{12}$")
CHAR_ID_RE = re.compile(r"^c_[0-9a-f]{12}$")
ITEM_ID_RE = re.compile(r"^[wgc]_[0-9a-f]{12}$")
CONTENT_ID_RE = re.compile(r"^ct_[0-9a-f]{12}$")
ARTIFACT_ID_RE = re.compile(r"^ar_[0-9a-f]{12}$")
EVENT_ULID_RE = re.compile(r"^ev_[0-9A-Z]{26}$")
EVENT_UUID_RE = re.compile(r"^ev_[0-9a-f]{32}$")


def normalize_pinyin(value: str) -> str:
    parts = value.strip().split()
    return " ".join(parts).lower()


def _hex12(payload: str) -> str:
    return hashlib.sha1(payload.encode("utf-8")).hexdigest()[:12]


def word_id(*, hanzi: str, pinyin: str) -> str:
    normalized = normalize_pinyin(pinyin)
    return f"w_{_hex12(f'word|{hanzi}|{normalized}')}"


def grammar_id(*, grammar_key: str) -> str:
    return f"g_{_hex12(f'grammar|{grammar_key}')}"


def char_id(*, character: str) -> str:
    return f"c_{_hex12(f'char|{character}')}"


def content_id(*, content_type: str, key: str) -> str:
    return f"ct_{_hex12(f'{content_type}|{key}')}"


def artifact_id(*, path: str) -> str:
    return f"ar_{_hex12(path)}"


def event_id_ulid() -> str:
    return f"ev_{ulid.new()}"


def is_word_id(value: str) -> bool:
    return bool(WORD_ID_RE.fullmatch(value))


def is_grammar_id(value: str) -> bool:
    return bool(GRAMMAR_ID_RE.fullmatch(value))


def is_char_id(value: str) -> bool:
    return bool(CHAR_ID_RE.fullmatch(value))


def is_item_id(value: str) -> bool:
    return bool(ITEM_ID_RE.fullmatch(value))


def is_event_id(value: str) -> bool:
    return bool(EVENT_ULID_RE.fullmatch(value) or EVENT_UUID_RE.fullmatch(value))


def is_content_id(value: str) -> bool:
    return bool(CONTENT_ID_RE.fullmatch(value))


def is_artifact_id(value: str) -> bool:
    return bool(ARTIFACT_ID_RE.fullmatch(value))
