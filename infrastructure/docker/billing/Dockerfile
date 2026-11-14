# Multi-stage build for Billing Service (port 9001)
FROM golang:1.23-alpine AS builder

WORKDIR /app

# No cgo dependencies: configuration comes from environment variables, so a
# static build avoids needing gcc/musl-dev in the builder image.
COPY billing_subscription/go/go.mod billing_subscription/go/go.sum ./
RUN go mod download

COPY billing_subscription/go/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /billing-service .

FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -u 1000 appuser

COPY --from=builder /billing-service /app/billing-service

USER appuser

EXPOSE 9001

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9001/health || exit 1

CMD ["./billing-service"]
