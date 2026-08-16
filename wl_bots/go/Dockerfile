# wl-bots standalone backend — multi-stage build.
# Build: docker build -t wl-bots .
# Run:   docker run -p 8471:8471 -e DATABASE_URL=... -e JWT_SECRET=... \
#          -e TWO_PARTY_GATE_URL=... -e WL_CLIENT_ID=... -e WL_LICENSE_KEY=... wl-bots
FROM golang:1.23-alpine AS builder
WORKDIR /src

# Cache deps. Copy only manifests first.
COPY go.mod go.sum ./
RUN go mod download

# Copy source (wl-bots + replaced wl-shared sibling).
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/wl-bots ./...

# ---------- runtime ----------
FROM alpine:3.19
RUN apk add --no-cache ca-certificates wget && \
    adduser -D -u 10001 wlbot
WORKDIR /app
COPY --from=builder /out/wl-bots /app/wl-bots
USER wlbot
EXPOSE 8471
ENV PORT=8471
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8471/health || exit 1
ENTRYPOINT ["/app/wl-bots"]
