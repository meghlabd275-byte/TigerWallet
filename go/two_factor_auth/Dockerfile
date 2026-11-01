# syntax=docker/dockerfile:1
#
# TigerWallet go/two_factor_auth
#
# This module replaces github.com/tigerwallet/sharedauth with ../sharedauth, so
# the build context must be ./go rather than this directory:
#
#   build:
#     context: ./go
#     dockerfile: two_factor_auth/Dockerfile
#
# Multi-stage build producing a static binary in a minimal runtime image.
# The healthcheck probes the service's own route so the orchestrator only
# routes traffic to a replica that can serve it.

FROM golang:1.25-alpine AS builder

WORKDIR /src

# Copy the module and the sibling module its replace directive points at,
# keeping the same relative layout so ../sharedauth still resolves.
COPY two_factor_auth/ ./two_factor_auth/
COPY sharedauth/ ./sharedauth/

WORKDIR /src/two_factor_auth

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags="-s -w" \
        -o /out/two-factor-auth .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget \
 && adduser -D -H -u 10001 appuser

WORKDIR /app
COPY --from=builder /out/two-factor-auth /app/two-factor-auth

USER appuser

EXPOSE 8446

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8446/health || exit 1

ENTRYPOINT ["/app/two-factor-auth"]
