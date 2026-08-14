# TigerWallet Admin API Dockerfile
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /admin-api ./main.go

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /admin-api .

EXPOSE 9093

CMD ["./admin-api"]
