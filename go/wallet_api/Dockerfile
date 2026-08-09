# Dockerfile for the TigerWallet Wallet API (Go)
# Multi-stage build for a minimal final image.
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Cache deps
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /wallet-api .

# Final image
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -h /app appuser

WORKDIR /app
COPY --from=builder /wallet-api /app/wallet-api

USER appuser

ENV WALLET_API_PORT=8443
EXPOSE 8443

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD wget -qO- http://localhost:8443/health || exit 1

ENTRYPOINT ["/app/wallet-api"]
