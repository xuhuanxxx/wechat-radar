# @lark-radar/api-contract

The single source of truth for the HTTP API between `apps/web` and `apps/data-service`.

## Workflow

1. **Edit** `openapi.yaml`. This is the source.
2. **Regenerate** consumer types:
   ```bash
   pnpm --filter @lark-radar/api-contract generate
   ```
3. **Commit** both `openapi.yaml` and the generated files (`generated/types.d.ts` and `apps/data-service/api/types.gen.go`). The generated artifacts are checked in so consumers don't need the codegen toolchain just to build.
4. CI runs `pnpm --filter @lark-radar/api-contract check`, which regenerates into a temp dir and diffs against the committed copies. If the spec changed without regenerating, this fails.

## Why two outputs, not one published JS lib?

- **TypeScript** consumers (`apps/web`) get type-only `.d.ts` so there's no runtime cost. They import via the `@lark-radar/api-contract` workspace alias.
- **Go** consumers (`apps/data-service`) get a `.gen.go` file dropped directly into the consuming module's `api/` package. Go cannot import from sibling workspace packages the way TS can; co-locating the generated file with its consumer is the idiomatic move.

## Tooling

- TS: [`openapi-typescript`](https://github.com/openapi-ts/openapi-typescript) — npm dep of this package.
- Go: [`oapi-codegen`](https://github.com/oapi-codegen/oapi-codegen) — install once with:
  ```bash
  go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
  ```

## What's in scope

Types-only. We do **not** generate server stubs, route handlers, or HTTP clients. The Go handlers in `apps/data-service/handlers/` keep their existing structure and just swap their handwritten response structs for the generated ones. The TypeScript side gets request/response shapes it can use to type `fetch` calls and React Query/SWR hooks.
