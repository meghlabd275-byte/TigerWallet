# syntax=docker/dockerfile:1
#
# TigerWallet go/cex_connector
#
# Multi-stage build producing a static binary in a minimal runtime image. The
# builder image matches the module's Go directive. The healthcheck probes the
# service's own route so the orchestrator only routes to a replica that serves.

FROM golang:1.25-alpine AS builder

WORKDIR /src

# Dependencies first so the module cache is reused across code changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags="-s -w" \
        -o /out/cex-connector .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget \
 && adduser -D -H -u 10001 appuser

WORKDIR /app
COPY --from=builder /out/cex-connector /app/cex-connector

USER appuser

EXPOSE 8090

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8090/health || exit 1

ENTRYPOINT ["/app/cex-connector"]
