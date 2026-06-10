#!/usr/bin/env bash
# Verify that committed generated artifacts match a fresh regeneration.
# Run this in CI to catch openapi.yaml drift from the generated outputs.
set -euo pipefail

cd "$(dirname "$0")/.."

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

cp generated/types.d.ts "$tmp/types.d.ts.committed"
cp ../../apps/data-service/api/types.gen.go "$tmp/types.gen.go.committed"

./scripts/generate.sh >/dev/null

diff -u "$tmp/types.d.ts.committed" generated/types.d.ts \
  || { echo "ERROR: generated/types.d.ts is stale. Run pnpm --filter @lark-radar/api-contract generate." >&2; exit 1; }

diff -u "$tmp/types.gen.go.committed" ../../apps/data-service/api/types.gen.go \
  || { echo "ERROR: apps/data-service/api/types.gen.go is stale. Run pnpm --filter @lark-radar/api-contract generate." >&2; exit 1; }

echo "Generated artifacts are in sync with openapi.yaml."
