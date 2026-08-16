# Standalone WL-MasterWallet backend Dockerfile.
# Runs INDEPENDENTLY in the WL client's own cloud — own PG, own signing, own DB.
# Multi-stage: golang:1.23-alpine -> alpine:3.19 (non-root).
FROM golang:1.23-alpine AS builder

WORKDIR /app

# The shared wl-shared module is referenced via a replace directive pointing at
# ../wl_shared/go, so it must be present in the build context.
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /wl-master-wallet .

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata wget && \
    adduser -D -h /app appuser

WORKDIR /app
COPY --from=builder /wl-master-wallet .

USER appuser
EXPOSE 8450

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD wget -qO- http://localhost:8450/health || exit 1

ENTRYPOINT ["./wl-master-wallet"]
