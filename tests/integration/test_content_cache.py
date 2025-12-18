import json
import os
import subprocess
import sys

from xuezh.core import ids, paths


def _run(env: dict[str, str], *args: str) -> dict:
    p = subprocess.run([sys.executable, "-m", "xuezh.cli", *args], env=env, capture_output=True, text=True)
    assert p.returncode == 0, p.stderr
    return json.loads(p.stdout)


def test_content_cache_put_get_idempotent(tmp_path):
    env = os.environ.copy()
    env["XUEZH_WORKSPACE_DIR"] = str(tmp_path)
    os.environ["XUEZH_WORKSPACE_DIR"] = str(tmp_path)

    input_path = tmp_path / "input.txt"
    input_path.write_text("hello", encoding="utf-8")

    out = _run(env, "content", "cache", "put", "--type", "story", "--key", "abc123", "--in", str(input_path), "--json")
    assert out["ok"] is True
    content_id = out["data"]["content_id"]
    assert content_id == ids.content_id(content_type="story", key="abc123")

    get_out = _run(env, "content", "cache", "get", "--type", "story", "--key", "abc123", "--json")
    assert get_out["ok"] is True
    artifact = get_out["artifacts"][0]
    cached_path = paths.resolve_in_workspace(artifact["path"])
    assert cached_path.read_text(encoding="utf-8") == "hello"

    # Idempotent: second put doesn't overwrite existing cache
    input_path.write_text("changed", encoding="utf-8")
    _run(env, "content", "cache", "put", "--type", "story", "--key", "abc123", "--in", str(input_path), "--json")
    assert cached_path.read_text(encoding="utf-8") == "hello"
