import json
import subprocess
import sys


def run(*args: str) -> dict:
    p = subprocess.run([sys.executable, "-m", "chlearn.cli", *args], capture_output=True, text=True)
    assert p.returncode == 0
    return json.loads(p.stdout)


def test_version_json_envelope():
    out = run("version", "--json")
    assert out["ok"] is True
    assert out["command"] == "version"
    assert "data" in out
    assert "artifacts" in out
