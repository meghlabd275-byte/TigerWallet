# This image must be built from the REPOSITORY ROOT, not from this directory.
#
# The module imports github.com/tigerwallet/chainsdk via a relative replace
# (../chainsdk). A module-local context cannot see that sibling directory, so
# `go mod download` fails with "replacement directory does not exist".
#
# Use the shared root-context Dockerfile instead:
#
#   docker build -f deployments/docker/Dockerfile.go-backend \
#     --build-arg TARGET_MODULE=master_wallet/backend -t tigerwallet-master-wallet .
#
# The compose files in this repository already pass the correct context,
# dockerfile and build args.

FROM golang:1.23-alpine AS builder
ARG TARGET_MODULE=master_wallet/backend
WORKDIR /src
COPY go/chainsdk/go.mod go/chainsdk/go.sum ./go/chainsdk/
COPY go/wallet_api/go.mod go/wallet_api/go.sum ./go/wallet_api/
COPY master_wallet/backend/go.mod master_wallet/backend/go.sum ./master_wallet/backend/
RUN cd "$TARGET_MODULE" && go mod download
COPY go/chainsdk ./go/chainsdk
COPY go/wallet_api ./go/wallet_api
COPY master_wallet/backend ./master_wallet/backend
RUN cd "$TARGET_MODULE" && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/backend .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -h /app appuser
COPY --from=builder /out/backend /usr/local/bin/backend
USER appuser
ENTRYPOINT ["/usr/local/bin/backend"]
