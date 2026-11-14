# syntax=docker/dockerfile:1
#
# TigerWallet institutional_custody
#
# Multi-stage build producing a static binary in a minimal runtime image.
# Every /api/v1/custody route requires CUSTODY_API_KEY, so the container
# refuses to start without it rather than serving unauthenticated withdrawals.

FROM golang:1.23-alpine AS builder

WORKDIR /src

# Dependencies first so the module cache is reused across code changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags="-s -w" \
        -o /out/custody-service .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget \
 && adduser -D -H -u 10001 appuser

WORKDIR /app
COPY --from=builder /out/custody-service /app/custody-service

USER appuser

EXPOSE 8097

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8097/health || exit 1

ENTRYPOINT ["/app/custody-service"]
