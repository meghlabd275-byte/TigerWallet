# wl-liquidity standalone backend — multi-stage build.
# Runs INDEPENDENTLY in the WL client's own cloud — own PG, own DB.
# Build: docker build -t wl-liquidity .
# Run:   docker run -p 8462:8462 -e DATABASE_URL=... -e JWT_SECRET=... \
#          -e TWO_PARTY_GATE_URL=... -e WL_CLIENT_ID=... -e WL_LICENSE_KEY=... wl-liquidity
FROM golang:1.23-alpine AS builder
WORKDIR /src

# Cache deps. Copy only manifests first.
COPY go.mod go.sum* ./
RUN go mod download

# Copy source (wl-liquidity + replaced wl-shared sibling).
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/wl-liquidity ./...

# ---------- runtime ----------
FROM alpine:3.19
RUN apk add --no-cache ca-certificates wget && \
    adduser -D -u 10001 wlliqa
WORKDIR /app
COPY --from=builder /out/wl-liquidity /app/wl-liquidity
USER wlliqa
EXPOSE 8462
ENV LIQ_PORT=8462
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8462/health || exit 1
ENTRYPOINT ["/app/wl-liquidity"]
