# License control plane Dockerfile.
# SuperAdmin kill-switch / feature-flag governance / two-party withdrawal gate.
# Multi-stage: golang:1.23-alpine -> alpine:3.19 (non-root).
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /license-service .

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -h /app appuser

WORKDIR /app
COPY --from=builder /license-service .

USER appuser
EXPOSE 8460

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD wget -qO- http://localhost:8460/health || exit 1

ENTRYPOINT ["./license-service"]
