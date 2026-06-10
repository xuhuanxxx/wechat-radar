# syntax=docker/dockerfile:1

# ─── Build Stage ───
FROM node:22-alpine AS builder

WORKDIR /app

# Install dependencies for native modules
RUN apk add --no-cache python3 make g++

# Copy package files
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./

# Install pnpm
RUN npm install -g pnpm

# Install dependencies
RUN pnpm install --frozen-lockfile

# Copy source code
COPY . .

# Build the Next.js app (standalone output)
ENV NEXT_TELEMETRY_DISABLED=1
RUN pnpm build

# ─── Production Stage ───
FROM node:22-alpine AS runner

WORKDIR /app

# Create non-root user
RUN addgroup --system --gid 1001 nodejs && \
    adduser --system --uid 1001 nextjs

# Copy standalone output from builder
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static
COPY --from=builder --chown=nextjs:nodejs /app/public ./public

# Switch to non-root user
USER nextjs

# Expose port (override at runtime with -e PORT=xxxx)
EXPOSE 8787

# Health check (uses PORT env var, defaults to 8787)
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD node -e "const p=process.env.PORT||8787;require('http').get('http://localhost:'+p+'/api/health',(r)=>r.statusCode===200?process.exit(0):process.exit(1))"

# Start the standalone server
# Default data service URL (override at runtime with -e DATA_API_URL=...)
ENV DATA_API_URL=""

ENV PORT=8787
ENV HOSTNAME="0.0.0.0"
CMD ["node", "server.js"]
