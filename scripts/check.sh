#!/usr/bin/env bash
set -euo pipefail

echo "== ruff =="
python -m ruff check .

echo "== mypy =="
python -m mypy src

echo "== pytest (no skips) =="
python -m pytest
