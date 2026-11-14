# syntax=docker/dockerfile:1
#
# TigerWallet analytics/go — event ingestion and metric plane.
#
# This is the write-side counterpart to go/analytics_service (the read-side
# dashboard API). It accepts product events and metric samples and exposes the
# aggregations built from them. The healthcheck probes the service's own route
# so the orchestrator only routes traffic to a replica that can serve it.

FROM golang:1.25-alpine AS builder

WORKDIR /src

# Dependencies first so the module cache is reused across code changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags="-s -w" \
        -o /out/analytics-event-plane ./cmd/analytics-service

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget \
 && adduser -D -H -u 10003 appuser

WORKDIR /app
COPY --from=builder /out/analytics-event-plane /app/analytics-event-plane

USER appuser

EXPOSE 9104

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:9104/health || exit 1

ENTRYPOINT ["/app/analytics-event-plane"]
