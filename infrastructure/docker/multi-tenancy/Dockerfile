# Multi-stage build for the Multi-Tenancy control-plane service (port 9002).
# Build context is the repository root:
#   docker build -f infrastructure/docker/multi-tenancy/Dockerfile .
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY multi_tenancy/go/go.mod multi_tenancy/go/go.sum ./
RUN go mod download

COPY multi_tenancy/go/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /multi-tenancy-service ./cmd

FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -u 1000 appuser

COPY --from=builder /multi-tenancy-service /app/multi-tenancy-service

USER appuser

EXPOSE 9002

HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9002/health || exit 1

CMD ["./multi-tenancy-service"]
