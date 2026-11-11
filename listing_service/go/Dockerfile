# syntax=docker/dockerfile:1
#
# TigerWallet listing_service/go (legacy, PostgreSQL-backed)
#
# Multi-stage build producing a static binary in a minimal runtime image.
# The healthcheck probes the service's own route, which answers 503 when the
# database is unreachable, so the orchestrator only routes to a working replica.

FROM golang:1.21-alpine AS builder

WORKDIR /src

# Dependencies first so the module cache is reused across code changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags="-s -w" \
        -o /out/listing-service .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget \
 && adduser -D -H -u 10001 appuser

WORKDIR /app
COPY --from=builder /out/listing-service /app/listing-service

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["/app/listing-service"]
