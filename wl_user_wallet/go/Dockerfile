# Standalone WL-UserWallet backend Dockerfile.
# Runs INDEPENDENTLY in the WL client's own environment.
# Multi-stage: golang:1.23-alpine -> alpine:3.19 (non-root).
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Cache deps
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /wl-user-wallet .

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -h /app appuser

WORKDIR /app
COPY --from=builder /wl-user-wallet .

USER appuser
EXPOSE 8443

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD wget -qO- http://localhost:8443/health || exit 1

ENTRYPOINT ["./wl-user-wallet"]
