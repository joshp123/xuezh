#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

WORKSPACE="$HOME/.local/share/xuezh/cram-local"
CORPUS="${XUEZH_HELLOCHINESE_CORPUS:-/Users/josh/Library/Mobile Documents/com~apple~CloudDocs/Full Pleco Import.txt}"
TRAVEL_CORPUS="${XUEZH_TRAVEL_CORPUS:-/Users/josh/Library/Mobile Documents/com~apple~CloudDocs/Travel Survival Pleco Import.txt}"
PLECO_BACKUP="${XUEZH_PLECO_BACKUP:-/Users/josh/Downloads/Pleco Flash Backup 260502.pqb}"
PORT="${XUEZH_WEB_PORT:-8765}"
AUDIO_VOICES="${XUEZH_AUDIO_VOICES:-zh-CN-XiaoxiaoNeural,zh-CN-XiaoyiNeural,zh-CN-YunxiNeural,zh-CN-YunyangNeural}"
AUDIO_RATES="${XUEZH_AUDIO_RATES:-zh-CN-XiaoxiaoNeural=-23%,zh-CN-XiaoyiNeural=-15%,zh-CN-YunxiNeural=-15%,zh-CN-YunyangNeural=-25%}"
AUDIO_CONCURRENCY="${XUEZH_AUDIO_CONCURRENCY:-8}"
AUDIO_REPLACE="${XUEZH_AUDIO_REPLACE:-0}"
PLECO_FORCE="${XUEZH_FORCE_PLECO_IMPORT:-0}"
CONFIG_HOME="$(mktemp -d "${TMPDIR:-/tmp}/xuezh-cram-config.XXXXXX")"
trap 'rm -rf "$CONFIG_HOME"' EXIT
mkdir -p "$CONFIG_HOME/xuezh"
workspace_toml="${WORKSPACE//\\/\\\\}"
workspace_toml="${workspace_toml//\"/\\\"}"
printf '[workspace]\ndir = "%s"\n' "$workspace_toml" > "$CONFIG_HOME/xuezh/config.toml"

run_xuezh() {
  XDG_CONFIG_HOME="$CONFIG_HOME" go run ./cmd/xuezh-go "$@"
}

run_xuezh_devenv() {
  XDG_CONFIG_HOME="$CONFIG_HOME" devenv shell -- go run ./cmd/xuezh-go "$@"
}

echo "workspace: $WORKSPACE"
echo "corpus:    $CORPUS"
echo "travel:    $TRAVEL_CORPUS"
echo "pleco:     $PLECO_BACKUP"
echo

run_xuezh hellochinese import \
  --path "$CORPUS" \
  --audio none \
  --json

run_xuezh travel import \
  --path "$TRAVEL_CORPUS" \
  --audio none \
  --json

if [ "$PLECO_FORCE" = "1" ] || [ ! -f "$WORKSPACE/db.sqlite3" ] || [ "$(sqlite3 "$WORKSPACE/db.sqlite3" "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='cram_pleco_scoring_settings'; SELECT COUNT(*) FROM cram_pleco_scoring_settings WHERE id=1;" 2>/dev/null | tail -1)" = "0" ]; then
  run_xuezh pleco score-import \
    --path "$PLECO_BACKUP" \
    --json
else
  echo "pleco:     already imported"
fi

if [ "${XUEZH_SKIP_AUDIO_BACKFILL:-0}" != "1" ]; then
  replace_args=()
  if [ "$AUDIO_REPLACE" = "1" ]; then
    replace_args=(--replace)
  fi
  run_xuezh_devenv cram audio-backfill \
    --source all \
    --voices "$AUDIO_VOICES" \
    --rates "$AUDIO_RATES" \
    --concurrency "$AUDIO_CONCURRENCY" \
    "${replace_args[@]}" \
    --json
fi

echo
echo "building web..."
if command -v pnpm >/dev/null 2>&1; then
  (cd web && pnpm install --silent && pnpm build)
else
  devenv shell -- sh -lc 'cd web && pnpm install --silent && pnpm build'
fi

echo
echo "open: http://127.0.0.1:$PORT"
echo

run_xuezh web serve --port "$PORT"
