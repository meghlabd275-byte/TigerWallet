# syntax=docker/dockerfile:1
# Multi-stage build for the TigerWallet White Label backend (Gin + pgx).

FROM golang:1.23-alpine AS builder
WORKDIR /src

# Cache module downloads first.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source and build a static-ish binary.
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/white-label-api .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates wget && \
    addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=builder /out/white-label-api /app/white-label-api

USER app
EXPOSE 8095

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- "http://127.0.0.1:8095/health" || exit 1

ENTRYPOINT ["/app/white-label-api"]
