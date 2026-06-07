# TIGERWALLET API GATEWAY
FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY backend_services/api_gateway/go.mod backend_services/api_gateway/go.sum ./
RUN go mod download

COPY backend_services/api_gateway/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /api-gateway

# RUNTIME
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /api-gateway .

ENV PORT=8080
EXPOSE 8080

CMD ["./api-gateway"]
