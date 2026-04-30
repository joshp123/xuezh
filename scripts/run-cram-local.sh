#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

WORKSPACE="${XUEZH_WORKSPACE_DIR:-$HOME/.local/share/xuezh/cram-local}"
CORPUS="${XUEZH_HELLOCHINESE_CORPUS:-/Users/josh/Documents/Codex/2026-04-24/can-you-use-the-hellochinese-plugin/snapshots/hellochinese_words.jsonl}"
PORT="${XUEZH_WEB_PORT:-8765}"

echo "workspace: $WORKSPACE"
echo "corpus:    $CORPUS"
echo

XUEZH_WORKSPACE_DIR="$WORKSPACE" \
  go run ./cmd/xuezh-go hellochinese import \
  --path "$CORPUS" \
  --audio none \
  --json

echo
echo "building web..."
(cd web && pnpm install --silent && pnpm build)

echo
echo "open: http://127.0.0.1:$PORT"
echo

XUEZH_WORKSPACE_DIR="$WORKSPACE" \
  go run ./cmd/xuezh-go web serve --port "$PORT"
