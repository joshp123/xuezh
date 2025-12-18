import pytest

from xuezh.core import events


def test_parse_items_from_list():
    items = events.parse_items(items="w_aaaaaaaaaaaa,g_bbbbbbbbbbbb", items_file=None)
    assert items == ["w_aaaaaaaaaaaa", "g_bbbbbbbbbbbb"]


def test_parse_items_from_file(tmp_path, monkeypatch):
    monkeypatch.setenv("XUEZH_WORKSPACE_DIR", str(tmp_path))
    items_path = tmp_path / "items.txt"
    items_path.write_text("w_aaaaaaaaaaaa\nc_cccccccccccc\n", encoding="utf-8")

    items = events.parse_items(items=None, items_file=str(items_path))
    assert items == ["w_aaaaaaaaaaaa", "c_cccccccccccc"]


def test_parse_items_rejects_invalid_id(tmp_path, monkeypatch):
    monkeypatch.setenv("XUEZH_WORKSPACE_DIR", str(tmp_path))
    items_path = tmp_path / "items.txt"
    items_path.write_text("bad_id\n", encoding="utf-8")

    with pytest.raises(ValueError):
        events.parse_items(items=None, items_file=str(items_path))
