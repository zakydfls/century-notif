#!/usr/bin/env sh
# Runs HTTP + MQ for Air hot-reload (stops both on rebuild/SIGINT).
set -e

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
HTTP="$ROOT/tmp/http.exe"
MQ="$ROOT/tmp/mq.exe"

cleanup() {
	if [ -n "${HTTP_PID:-}" ]; then kill "$HTTP_PID" 2>/dev/null || true; fi
	if [ -n "${MQ_PID:-}" ]; then kill "$MQ_PID" 2>/dev/null || true; fi
	wait 2>/dev/null || true
}
trap cleanup INT TERM

"$HTTP" &
HTTP_PID=$!
"$MQ" &
MQ_PID=$!

wait "$HTTP_PID" "$MQ_PID"
