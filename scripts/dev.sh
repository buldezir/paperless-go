#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

BACKEND_PID=""
FRONTEND_PID=""

kill_tree() {
  local pid="$1"
  if [[ -z "$pid" ]] || ! kill -0 "$pid" 2>/dev/null; then
    return 0
  fi
  # Kill children first (e.g. go-run binary, vite under npm).
  local child
  for child in $(pgrep -P "$pid" 2>/dev/null || true); do
    kill_tree "$child"
  done
  kill "$pid" 2>/dev/null || true
  wait "$pid" 2>/dev/null || true
}

cleanup() {
  local code=$?
  trap - EXIT INT TERM
  echo ""
  echo "Stopping dev servers..."
  kill_tree "$FRONTEND_PID"
  kill_tree "$BACKEND_PID"
  exit "$code"
}

trap cleanup EXIT INT TERM

if [[ ! -f .env ]]; then
  echo "warning: .env not found; copy from .env.example if needed" >&2
fi

prefix() {
  local name="$1"
  local color="$2"
  local reset=$'\033[0m'
  while IFS= read -r line || [[ -n "$line" ]]; do
    printf '%b[%s]%b %s\n' "$color" "$name" "$reset" "$line"
  done
}

echo "Starting backend on http://127.0.0.1:8090"
(cd "$ROOT/backend" && go run . serve --http=127.0.0.1:8090) \
  > >(prefix "backend" $'\033[36m') 2>&1 &
BACKEND_PID=$!

echo "Starting frontend on http://127.0.0.1:5173"
(cd "$ROOT/frontend" && npm run dev -- --host 127.0.0.1 --port 5173) \
  > >(prefix "frontend" $'\033[33m') 2>&1 &
FRONTEND_PID=$!

echo ""
echo "Dev servers running. Press Ctrl+C to stop."
echo "  Backend:  http://127.0.0.1:8090"
echo "  Frontend: http://127.0.0.1:5173"
echo ""

while kill -0 "$BACKEND_PID" 2>/dev/null && kill -0 "$FRONTEND_PID" 2>/dev/null; do
  sleep 1
done

if ! kill -0 "$BACKEND_PID" 2>/dev/null; then
  wait "$BACKEND_PID" || true
  echo "backend exited" >&2
  exit 1
fi
if ! kill -0 "$FRONTEND_PID" 2>/dev/null; then
  wait "$FRONTEND_PID" || true
  echo "frontend exited" >&2
  exit 1
fi
