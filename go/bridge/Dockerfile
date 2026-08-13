FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bridge-service .
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata && adduser -D -h /app appuser
WORKDIR /app
COPY --from=builder /bridge-service /app/bridge-service
USER appuser
EXPOSE 8470
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD wget -qO- http://localhost:8470/health || exit 1
ENTRYPOINT ["/app/bridge-service"]
