import json
import subprocess
import sys


def run(*args: str) -> dict:
    p = subprocess.run([sys.executable, "-m", "xuezh.cli", *args], capture_output=True, text=True)
    assert p.returncode == 0
    return json.loads(p.stdout)


def test_doctor_includes_whisper_check():
    out = run("doctor", "--json")
    checks = out["data"]["checks"]
    names = {check["name"] for check in checks}
    assert "tool.whisper" in names
