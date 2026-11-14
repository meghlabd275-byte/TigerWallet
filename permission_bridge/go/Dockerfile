FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /permission-bridge ./cmd/tiger-permission-bridge

FROM alpine:latest
RUN apk --no-cache add ca-certificates
RUN adduser -D -u 10003 bridge
WORKDIR /app
COPY --from=builder /permission-bridge .
USER bridge
EXPOSE 9007
CMD ["./permission-bridge"]
