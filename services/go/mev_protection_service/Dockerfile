# syntax=docker/dockerfile:1
#
# TigerWallet mev_protection_service
#
# Builds real Flashbots-protected bundles and maintains the address blocklist.
# MEV_API_KEY is required: the blocklist is a censorship surface, so the
# container refuses to start without it.

FROM golang:1.22-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags="-s -w" \
        -o /out/mev-protection-service .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget \
 && adduser -D -H -u 10001 appuser

WORKDIR /app
COPY --from=builder /out/mev-protection-service /app/mev-protection-service

USER appuser

EXPOSE 8089

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8089/health || exit 1

ENTRYPOINT ["/app/mev-protection-service"]
