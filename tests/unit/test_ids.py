from xuezh.core import ids


def test_normalize_pinyin():
    assert ids.normalize_pinyin("  Ni3   hao ") == "ni3 hao"


def test_word_id_deterministic_and_valid():
    first = ids.word_id(hanzi="你好", pinyin="Ni3   hao")
    second = ids.word_id(hanzi="你好", pinyin="ni3 hao")
    assert first == second
    assert ids.is_word_id(first)
    assert ids.is_item_id(first)


def test_grammar_id_valid():
    value = ids.grammar_id(grammar_key="hsk1-lesson3")
    assert ids.is_grammar_id(value)
    assert ids.is_item_id(value)


def test_char_id_valid():
    value = ids.char_id(character="你")
    assert ids.is_char_id(value)
    assert ids.is_item_id(value)


def test_content_id_valid():
    value = ids.content_id(content_type="story", key="hsk1-001")
    assert ids.is_content_id(value)


def test_artifact_id_valid():
    value = ids.artifact_id(path="artifacts/audio/voice.ogg")
    assert ids.is_artifact_id(value)


def test_event_id_ulid_format():
    value = ids.event_id_ulid()
    assert ids.is_event_id(value)
