# wl-card standalone backend — multi-stage build.
# Runs INDEPENDENTLY in the WL client's own cloud — own PG, own DB.
# Build: docker build -t wl-card .
# Run:   docker run -p 8463:8463 -e DATABASE_URL=... -e JWT_SECRET=... \
#          -e CARD_ENC_KEY=... -e TWO_PARTY_GATE_URL=... -e WL_CLIENT_ID=... \
#          -e WL_LICENSE_KEY=... wl-card
FROM golang:1.23-alpine AS builder
WORKDIR /src

# Cache deps. Copy only manifests first.
COPY go.mod go.sum* ./
RUN go mod download

# Copy source (wl-card + replaced wl-shared sibling).
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/wl-card ./...

# ---------- runtime ----------
FROM alpine:3.19
RUN apk add --no-cache ca-certificates wget && \
    adduser -D -u 10001 wlcard
WORKDIR /app
COPY --from=builder /out/wl-card /app/wl-card
USER wlcard
EXPOSE 8463
ENV CARD_PORT=8463
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8463/health || exit 1
ENTRYPOINT ["/app/wl-card"]
