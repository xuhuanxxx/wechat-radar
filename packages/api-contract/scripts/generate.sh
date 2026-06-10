#!/usr/bin/env bash
# Regenerate TypeScript + Go types from openapi.yaml.
#
# Prereqs:
#   - pnpm install (provides openapi-typescript locally)
#   - go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
#     (or use `go run` via tools.go if you prefer pinning)
set -euo pipefail

cd "$(dirname "$0")/.."

mkdir -p generated

echo "==> openapi-typescript → generated/types.d.ts"
pnpm exec openapi-typescript openapi.yaml -o generated/types.d.ts

echo "==> oapi-codegen → ../../apps/data-service/api/types.gen.go"
if ! command -v oapi-codegen >/dev/null 2>&1; then
  echo "oapi-codegen not on PATH. Install with:" >&2
  echo "  go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest" >&2
  exit 1
fi
oapi-codegen --config oapi-codegen.yaml openapi.yaml

echo "Generation complete."
