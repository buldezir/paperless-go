#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

stage() {
  echo ""
  echo "==> $1"
}

fail() {
  echo ""
  echo "FAILED: $1" >&2
  exit 1
}

stage "Unit tests (excluding e2e)"
(
  cd backend
  # shellcheck disable=SC2046
  go test $(go list ./... | grep -v '/e2e$') -count=1
) || fail "unit tests"

stage "API e2e"
(
  cd backend
  go test ./e2e/ -count=1 -timeout 10m
) || fail "API e2e"

stage "Frontend build + Playwright e2e"
(
  cd frontend
  export PLAYWRIGHT_SKIP_VALIDATE_HOST_REQUIREMENTS="${PLAYWRIGHT_SKIP_VALIDATE_HOST_REQUIREMENTS:-1}"
  npm run test:e2e
) || fail "browser e2e"

echo ""
echo "All verification stages passed."
