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

# Two flavors of check:
#   check_ok PATH     — response is JSON AND ok=true (the common case)
#   check_shape PATH  — response is JSON with an `ok` boolean, regardless
#                       of its value. Used for endpoints whose `ok` is
#                       legitimately false in stripped environments
#                       (e.g. /api/doctor without lark-cli installed).

check_ok() {
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

check_shape() {
  local path=$1
  local body status
  if ! body=$(curl -fsS "$BASE$path" 2>&1); then
    echo "  ✗ GET $path — curl failed: $body"
    fail=$((fail + 1))
    return
  fi
  if ! status=$(printf '%s' "$body" | jq -r '.ok' 2>/dev/null); then
    echo "  ✗ GET $path — response is not JSON with .ok"
    echo "    body: $(printf '%s' "$body" | head -c 200)"
    fail=$((fail + 1))
    return
  fi
  if [ "$status" != "true" ] && [ "$status" != "false" ]; then
    echo "  ✗ GET $path — .ok is not a boolean (got: $status)"
    fail=$((fail + 1))
    return
  fi
  echo "  ✓ GET $path (ok=$status)"
  pass=$((pass + 1))
}

# Health + config — must work even on an empty database
check_ok /api/health
check_ok /api/config

# /api/doctor's ok is false in environments without lark-cli installed
# (e.g. CI runners). Only verify the response is well-formed JSON.
check_shape /api/doctor

# Read-only endpoints over an empty database — should return empty
# arrays with ok=true, NOT 500
check_ok /api/sessions
check_ok "/api/stats?range=7d"
check_ok /api/intelligence
check_ok /api/groups
check_ok /api/mentions
check_ok "/api/mentions/stats"
check_ok /api/links
check_ok /api/favorites
check_ok /api/topics
check_ok /api/new-messages

echo
echo "smoke-test: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
