# =============================================================================
# nexus-orchestrator — Multi-stage Docker image
# =============================================================================
# Stage 1: build nexus-daemon (requires CGO for go-sqlite3)
# Stage 2: minimal Alpine runtime
#
# Build args (injected by CI / make docker-build):
#   VERSION    — semver tag, e.g. 0.10.0-rc.1
#   COMMIT     — short git SHA
#   BUILD_DATE — RFC-3339 timestamp
#
# Runtime env vars:
#   NEXUS_LISTEN_ADDR   HTTP API  (default: 0.0.0.0:63987)
#   NEXUS_MCP_ADDR      MCP JSON-RPC  (default: 0.0.0.0:63988)
#   NEXUS_DB_PATH       SQLite file   (default: /data/nexus.db)
#   NEXUS_SCAN_INTERVAL agent scan interval, e.g. 30s
# =============================================================================

ARG GO_VERSION=1.26
ARG ALPINE_VERSION=3.21

# ---------------------------------------------------------------------------
# Stage 1 — builder
# ---------------------------------------------------------------------------
FROM golang:${GO_VERSION}-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=1 go build -trimpath \
      -ldflags "-s -w \
        -X 'main.version=${VERSION}' \
        -X 'main.commit=${COMMIT}' \
        -X 'main.buildDate=${BUILD_DATE}'" \
      -o /out/nexus-daemon ./cmd/nexus-daemon/... && \
    CGO_ENABLED=0 go build -trimpath \
      -ldflags "-s -w \
        -X 'main.version=${VERSION}' \
        -X 'main.commit=${COMMIT}' \
        -X 'main.buildDate=${BUILD_DATE}'" \
      -o /out/nexus-cli    ./cmd/nexus-cli/... && \
    CGO_ENABLED=0 go build -trimpath \
      -ldflags "-s -w \
        -X 'main.version=${VERSION}' \
        -X 'main.commit=${COMMIT}' \
        -X 'main.buildDate=${BUILD_DATE}'" \
      -o /out/nexus-submit ./cmd/nexus-submit/...

# ---------------------------------------------------------------------------
# Stage 2 — runtime (Alpine for libc/sqlite compat)
# ---------------------------------------------------------------------------
FROM alpine:${ALPINE_VERSION}

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /out/nexus-daemon  /usr/local/bin/nexus-daemon
COPY --from=builder /out/nexus-cli     /usr/local/bin/nexus-cli
COPY --from=builder /out/nexus-submit  /usr/local/bin/nexus-submit

# Data directory for SQLite database and project scans
VOLUME ["/data"]

# HTTP API + MCP JSON-RPC
EXPOSE 63987 63988

# In containers listen on all interfaces by default
ENV NEXUS_LISTEN_ADDR=0.0.0.0:63987 \
    NEXUS_MCP_ADDR=0.0.0.0:63988 \
    NEXUS_DB_PATH=/data/nexus.db

ENTRYPOINT ["nexus-daemon"]
