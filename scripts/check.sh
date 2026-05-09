#!/usr/bin/env bash
set -euo pipefail

echo "== go test =="
go test ./...

echo "== web offline service worker test =="
(cd web && pnpm run test:offline-sw)

echo "== web http timeout test =="
(cd web && pnpm run test:http)

echo "== web offline sync rule test =="
(cd web && pnpm run test:offline-sync)

echo "== web vite config test =="
(cd web && pnpm run test:vite-config)

echo "== web build =="
(cd web && pnpm build)
