# MasterWallet backend — multi-stage build (Go -> minimal Alpine runtime).
# Produces a single binary that replaces the 4 redundant broken Go backends.
FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /master-wallet .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /master-wallet /usr/local/bin/master-wallet
EXPOSE 8450
ENTRYPOINT ["master-wallet"]
