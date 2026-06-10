# syntax=docker/dockerfile:1
#
# Build context: repo root.
# Build command: docker build -f apps/web/Dockerfile -t lark-radar-web .
#
# This Dockerfile builds @lark-radar/web from the pnpm workspace, then
# packages only the standalone Next.js server output.

# ─── Build Stage ───
FROM node:22-alpine AS builder

WORKDIR /repo

RUN apk add --no-cache python3 make g++ && npm install -g pnpm

# Copy workspace manifests first to maximize cache hits on dep installs.
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/web/package.json apps/web/package.json
COPY packages/api-contract/package.json packages/api-contract/package.json

RUN pnpm install --frozen-lockfile --filter @lark-radar/web...

# Copy the web app + the workspace deps it imports from (api-contract
# ships only generated TypeScript declarations — no runtime — but they
# must be present at type-check time, which is part of `next build`).
COPY apps/web apps/web
COPY packages/api-contract packages/api-contract

ENV NEXT_TELEMETRY_DISABLED=1
RUN pnpm --filter @lark-radar/web build

# ─── Production Stage ───
FROM node:22-alpine AS runner

WORKDIR /app

RUN addgroup --system --gid 1001 nodejs && \
    adduser --system --uid 1001 nextjs

# Next.js standalone output: apps/web/.next/standalone is a self-contained
# Node project with server.js at its root and a nested apps/web/ tree.
COPY --from=builder --chown=nextjs:nodejs /repo/apps/web/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /repo/apps/web/.next/static ./apps/web/.next/static
COPY --from=builder --chown=nextjs:nodejs /repo/apps/web/public ./apps/web/public

USER nextjs

EXPOSE 8787

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD node -e "const p=process.env.PORT||8787;require('http').get('http://localhost:'+p+'/api/health',(r)=>r.statusCode===200?process.exit(0):process.exit(1))"

# Default data service URL (override at runtime with -e DATA_API_URL=...)
ENV DATA_API_URL=""

ENV PORT=8787
ENV HOSTNAME="0.0.0.0"
CMD ["node", "apps/web/server.js"]
