# White-Label Admin Panel backend — standalone, self-hosted by the WL client.
# Runs INDEPENDENTLY in the WL client's own cloud: own PG, own Redis, own DB,
# fail-closed license gate + heartbeat to the SuperAdmin control plane.
#
# Build:  docker build -t wl-admin .
# Run:    docker run -p 8461:8461 \
#           -e DATABASE_URL=... -e JWT_SECRET=... -e REDIS_ADDR=... \
#           -e TWO_PARTY_GATE_URL=... -e TWO_PARTY_GATE_TOKEN=... \
#           -e WL_CLIENT_ID=... -e LICENSE_KEY=... wl-admin
FROM golang:1.23-alpine AS builder
WORKDIR /src

# The shared wl-shared module is referenced via a replace directive pointing at
# ../../wl_shared/go, so it must be present in the build context (build with the
# repository root as context, or populate the sibling into the context).
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/wl-admin ./...

# ---------- runtime ----------
FROM alpine:3.19
RUN apk add --no-cache ca-certificates wget && \
    adduser -D -u 10002 wladmin
WORKDIR /app
COPY --from=builder /out/wl-admin /app/wl-admin
USER wladmin
EXPOSE 8456
ENV SERVER_PORT=8456
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8456/health || exit 1
ENTRYPOINT ["/app/wl-admin"]