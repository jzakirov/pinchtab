# Stage 1: Build the React dashboard with Bun.
# The compiled assets are copied into the Go embed directory in stage 2.
FROM oven/bun:1 AS dashboard
WORKDIR /build
COPY dashboard/package.json dashboard/bun.lock ./
RUN bun install --frozen-lockfile
COPY dashboard/ .
RUN bun run build

# Stage 2: Compile the Go binary.
# Dashboard dist is embedded into the binary via Go's embed package.
# The rename to dashboard.html avoids a conflict with the Go embed server's
# own index route — the dashboard handler serves it explicitly.
FROM golang:1.26-alpine3.21 AS builder
RUN apk add --no-cache git
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=dashboard /build/dist/ internal/dashboard/dashboard/
RUN mv internal/dashboard/dashboard/index.html internal/dashboard/dashboard/dashboard.html
RUN go build -ldflags="-s -w" -o pinchtab ./cmd/pinchtab

# Stage 3: Minimal runtime image with Chromium.
# Only the compiled binary and entrypoint script are copied in.
FROM alpine:3.21

LABEL org.opencontainers.image.source="https://github.com/pinchtab/pinchtab"
LABEL org.opencontainers.image.description="High-performance browser automation bridge"

# Chromium and its runtime dependencies for headless operation
RUN apk add --no-cache \
    chromium \
    nss \
    freetype \
    harfbuzz \
    ca-certificates \
    ttf-freefont \
    dumb-init

# Non-root user; /data is the persistent volume mount point
RUN adduser -D -h /data -g '' pinchtab && \
    mkdir -p /data && \
    chown pinchtab:pinchtab /data

COPY --from=builder /build/pinchtab /usr/local/bin/pinchtab
COPY --chmod=0755 docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

USER pinchtab
WORKDIR /data

# HOME and XDG_CONFIG_HOME point into the persistent volume so config
# and Chrome profiles survive container restarts.
ENV HOME=/data \
    XDG_CONFIG_HOME=/data/.config

EXPOSE 9867

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -q -O /dev/null http://localhost:9867/health || exit 1

ENTRYPOINT ["/usr/bin/dumb-init", "--"]
CMD ["/usr/local/bin/docker-entrypoint.sh", "pinchtab"]
