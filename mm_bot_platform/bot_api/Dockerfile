FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bot-api .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata && adduser -D -h /app appuser
WORKDIR /app
COPY --from=builder /bot-api /app/bot-api
USER appuser
EXPOSE 8471
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD wget -qO- http://localhost:8471/api/v1/health || exit 1
ENTRYPOINT ["/app/bot-api"]
