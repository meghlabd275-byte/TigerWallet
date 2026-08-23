# Kill-switch control plane Dockerfile.
# Emergency halt/resume for the whole platform, per WL client, per product,
# or per fetcher. PostgreSQL state + Redis propagation, SuperAdmin JWT only.
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /kill-switch .

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -h /app appuser

WORKDIR /app
COPY --from=builder /kill-switch .

USER appuser
EXPOSE 8469

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD wget -qO- http://localhost:8469/health || exit 1

ENTRYPOINT ["./kill-switch"]
