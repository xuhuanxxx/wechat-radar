#!/usr/bin/env bash
# Smoke-test the running stack end-to-end.
#
# Hits a representative slice of /api/* through whatever URL the caller
# passes in (web proxy in CI, or directly at the Go service for a local
# check) and asserts every response is JSON with `ok: true`.
#
# Usage:
#   scripts/smoke-test.sh                       # default: http://localhost:8787
#   scripts/smoke-test.sh http://localhost:3001 # through the web proxy
#   scripts/smoke-test.sh http://localhost:3456 # directly at the Go service
#
# Prereqs: curl, jq. Assumes the target is already running and reachable.

set -euo pipefail

BASE="${1:-http://localhost:8787}"
echo "smoke-test target: $BASE"
echo

pass=0
fail=0

check_get() {
  local path=$1
  local body status
  if ! body=$(curl -fsS "$BASE$path" 2>&1); then
    echo "  ✗ GET $path — curl failed: $body"
    fail=$((fail + 1))
    return
  fi
  if ! status=$(printf '%s' "$body" | jq -r '.ok' 2>/dev/null); then
    echo "  ✗ GET $path — response is not JSON"
    echo "    body: $(printf '%s' "$body" | head -c 200)"
    fail=$((fail + 1))
    return
  fi
  if [ "$status" != "true" ]; then
    echo "  ✗ GET $path — ok=$status"
    echo "    body: $(printf '%s' "$body" | head -c 200)"
    fail=$((fail + 1))
    return
  fi
  echo "  ✓ GET $path"
  pass=$((pass + 1))
}

# Health + config — must work even on an empty database
check_get /api/health
check_get /api/doctor
check_get /api/config

# Read-only endpoints over an empty database — should return empty
# arrays with ok=true, NOT 500
check_get /api/sessions
check_get "/api/stats?range=7d"
check_get /api/intelligence
check_get /api/groups
check_get /api/mentions
check_get "/api/mentions/stats"
check_get /api/links
check_get /api/favorites
check_get /api/topics
check_get /api/new-messages

echo
echo "smoke-test: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
