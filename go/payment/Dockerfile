# syntax=docker/dockerfile:1
#
# TigerWallet go/payment
#
# Multi-stage build producing a static binary in a minimal runtime image.
# The healthcheck probes the service's own route so the orchestrator only
# routes traffic to a replica that can serve it.

FROM golang:1.21-alpine AS builder

WORKDIR /src

# Dependencies first so the module cache is reused across code changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags="-s -w" \
        -o /out/payment-service .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget \
 && adduser -D -H -u 10001 appuser

WORKDIR /app
COPY --from=builder /out/payment-service /app/payment-service

USER appuser

EXPOSE 8096

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8096/health || exit 1

ENTRYPOINT ["/app/payment-service"]
