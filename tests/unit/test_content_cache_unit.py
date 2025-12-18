from __future__ import annotations

from pathlib import Path

import pytest

from xuezh.core import content, paths


def test_put_content_requires_valid_type(tmp_path, monkeypatch):
    monkeypatch.setenv("XUEZH_WORKSPACE_DIR", str(tmp_path))
    input_path = tmp_path / "input.txt"
    input_path.write_text("hello", encoding="utf-8")

    with pytest.raises(ValueError):
        content.put_content(content_type="unknown", key="k", in_path=str(input_path))


def test_put_content_stores_under_workspace(tmp_path, monkeypatch):
    monkeypatch.setenv("XUEZH_WORKSPACE_DIR", str(tmp_path))
    input_path = tmp_path / "input.txt"
    input_path.write_text("hello", encoding="utf-8")

    result = content.put_content(content_type="story", key="k1", in_path=str(input_path))
    artifact_path = paths.resolve_in_workspace(result.artifacts[0].path)
    assert artifact_path.exists()
    assert str(artifact_path).startswith(str(Path(tmp_path)))
