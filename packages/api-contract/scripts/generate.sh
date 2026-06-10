#!/usr/bin/env bash
# Regenerate TypeScript + Go types from openapi.yaml.
#
# Prereqs:
#   - pnpm install (provides openapi-typescript locally)
#   - go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
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

# Post-process: oapi-codegen lowercases common Go acronyms (Ok/Id/Url) when
# converting snake_case JSON to CamelCase fields. We rewrite them back to
# idiomatic Go (OK/ID/URL) so handler code reads naturally and so the new
# api.* types are drop-in compatible with the existing handwritten models.
GEN_FILE="../../apps/data-service/api/types.gen.go"
echo "==> gofmt -r fixups on ${GEN_FILE}"
gofmt -r 'Ok -> OK' -w "$GEN_FILE"
gofmt -r 'Id -> ID' -w "$GEN_FILE"
gofmt -r 'Url -> URL' -w "$GEN_FILE"
# Compound names that the per-token rewrite missed:
gofmt -r 'Ids -> IDs' -w "$GEN_FILE"
gofmt -r 'GroupIds -> GroupIDs' -w "$GEN_FILE"
gofmt -r 'ChatroomId  -> ChatroomID'  -w "$GEN_FILE"
gofmt -r 'ChatroomIds -> ChatroomIDs' -w "$GEN_FILE"
gofmt -r 'MessageId   -> MessageID'   -w "$GEN_FILE"
gofmt -r 'MessageIds  -> MessageIDs'  -w "$GEN_FILE"
gofmt -r 'LocalId     -> LocalID'     -w "$GEN_FILE"
gofmt -r 'ChatId      -> ChatID'      -w "$GEN_FILE"
gofmt -r 'ChatIds     -> ChatIDs'     -w "$GEN_FILE"
gofmt -r 'SenderId    -> SenderID'    -w "$GEN_FILE"
gofmt -r 'IdType      -> IDType'      -w "$GEN_FILE"
gofmt -r 'ParentId    -> ParentID'    -w "$GEN_FILE"
gofmt -r 'ThreadId    -> ThreadID'    -w "$GEN_FILE"

echo "Generation complete."
