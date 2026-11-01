# syntax=docker/dockerfile:1
#
# TigerWallet go/portfolio_tracker
#
# This module has no external dependencies, so there is no go.sum to copy and
# no module download step.
#
# Multi-stage build producing a static binary in a minimal runtime image.
# The healthcheck probes the service's own route so the orchestrator only
# routes traffic to a replica that can serve it.

FROM golang:1.22-alpine AS builder

WORKDIR /src

COPY go.mod ./
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags="-s -w" \
        -o /out/portfolio-tracker .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget \
 && adduser -D -H -u 10001 appuser

WORKDIR /app
COPY --from=builder /out/portfolio-tracker /app/portfolio-tracker

USER appuser

EXPOSE 8084

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8084/health || exit 1

ENTRYPOINT ["/app/portfolio-tracker"]
