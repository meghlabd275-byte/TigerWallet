# syntax=docker/dockerfile:1
#
# TigerWallet account_abstraction/go_api
#
# Multi-stage build producing a static binary in a minimal runtime image.
# The healthcheck probes the service's own route so the orchestrator only
# routes traffic to a replica that can serve it.

FROM golang:1.23-alpine AS builder

WORKDIR /src

# Dependencies first so the module cache is reused across code changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags="-s -w" \
        -o /out/aa-bundler ./cmd

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget \
 && adduser -D -H -u 10001 appuser

WORKDIR /app
COPY --from=builder /out/aa-bundler /app/aa-bundler

USER appuser

EXPOSE 8081

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8081/health || exit 1

ENTRYPOINT ["/app/aa-bundler"]
