#!/usr/bin/env bash
set -euo pipefail

echo "== go test =="
go test ./...

echo "== web offline service worker test =="
(cd web && pnpm run test:offline-sw)

echo "== web build =="
(cd web && pnpm build)
