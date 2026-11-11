# syntax=docker/dockerfile:1
#
# TigerWallet go/api_gateway
#
# Multi-stage build producing a static binary in a minimal runtime image. The
# builder image matches the module's Go directive. The healthcheck probes the
# service's own route so the orchestrator only routes to a replica that serves.

FROM golang:1.22-alpine AS builder

WORKDIR /src

# Dependencies first so the module cache is reused across code changes.
COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags="-s -w" \
        -o /out/api-gateway .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget \
 && adduser -D -H -u 10001 appuser

WORKDIR /app
COPY --from=builder /out/api-gateway /app/api-gateway

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["/app/api-gateway"]
