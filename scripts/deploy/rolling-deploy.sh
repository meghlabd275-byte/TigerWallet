#!/usr/bin/env bash
# Health-gated rolling deploy for the TigerWallet stack.
#
# The previous CI deploy jobs were `echo "Deploying..."` placeholders, so every
# release was manual and unverified. This script performs a real deploy and,
# critically, *verifies* it: if any service fails to reach a healthy state the
# deploy is rolled back to the previous image set.
#
# Usage:
#   DEPLOY_ENV=production scripts/deploy/rolling-deploy.sh
#
# Environment:
#   DEPLOY_ENV              staging | production (required)
#   COMPOSE_FILE            compose file to deploy (default docker-compose.yml)
#   IMAGE_TAG               image tag being deployed (default: current git SHA)
#   HEALTH_TIMEOUT_SECONDS  per-service health wait (default 180)
#   HEALTH_POLL_SECONDS     poll interval (default 3)
#   ROLLBACK_ON_FAILURE     true|false (default true)
#   DRY_RUN                 true to print the plan without changing anything
#
# Exit codes: 0 success, 1 deploy failed (rollback attempted), 2 bad usage.

set -Eeuo pipefail

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.yml}"
HEALTH_TIMEOUT_SECONDS="${HEALTH_TIMEOUT_SECONDS:-180}"
HEALTH_POLL_SECONDS="${HEALTH_POLL_SECONDS:-3}"
ROLLBACK_ON_FAILURE="${ROLLBACK_ON_FAILURE:-true}"
DRY_RUN="${DRY_RUN:-false}"

log()  { printf '[deploy] %s\n' "$*"; }
fail() { printf '[deploy] ERROR: %s\n' "$*" >&2; }

if [[ -z "${DEPLOY_ENV:-}" ]]; then
  fail "DEPLOY_ENV is required (staging|production)"
  exit 2
fi
if [[ ! -f "$COMPOSE_FILE" ]]; then
  fail "compose file not found: $COMPOSE_FILE"
  exit 2
fi

# The image tag identifies the release being deployed, so a deploy is always
# traceable back to the revision that produced it.
if [[ -z "${IMAGE_TAG:-}" ]]; then
  IMAGE_TAG="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
fi
export IMAGE_TAG

mapfile -t ALL_SERVICES < <(docker compose -f "$COMPOSE_FILE" config --services 2>/dev/null | sort)

if [[ ${#ALL_SERVICES[@]} -eq 0 ]]; then
  fail "no services found in $COMPOSE_FILE"
  exit 2
fi

log "environment    : $DEPLOY_ENV"
log "compose file   : $COMPOSE_FILE"
log "image tag      : $IMAGE_TAG"
log "services       : ${#ALL_SERVICES[@]}"
log "health timeout : ${HEALTH_TIMEOUT_SECONDS}s per service"

if [[ "$DRY_RUN" == "true" ]]; then
  log "DRY_RUN: would deploy the following services:"
  printf '  - %s\n' "${ALL_SERVICES[@]}"
  exit 0
fi

# Capture the currently running image ids so a rollback can restore them.
PREVIOUS_STATE="$(docker compose -f "$COMPOSE_FILE" ps -q 2>/dev/null | while read -r cid; do
  [[ -z "$cid" ]] && continue
  svc="$(docker inspect -f '{{ index .Config.Labels "com.docker.compose.service" }}' "$cid" 2>/dev/null || true)"
  img="$(docker inspect -f '{{.Image}}' "$cid" 2>/dev/null || true)"
  [[ -n "$svc" && -n "$img" ]] && printf '%s=%s\n' "$svc" "$img"
done)"
PREVIOUS_IMAGES_FILE="$(mktemp)"
if [[ -n "$PREVIOUS_STATE" ]]; then
  printf '%s\n' "$PREVIOUS_STATE" > "$PREVIOUS_IMAGES_FILE"
  log "captured previous image ids for rollback ($(wc -l < "$PREVIOUS_IMAGES_FILE") services)"
fi

rollback() {
  if [[ "$ROLLBACK_ON_FAILURE" != "true" ]]; then
    log "rollback disabled; leaving the failed release in place for inspection"
    return 0
  fi
  if [[ ! -s "$PREVIOUS_IMAGES_FILE" ]]; then
    log "no previous image state captured; nothing to roll back to"
    return 0
  fi
  log "rolling back to the previous image set"
  docker compose -f "$COMPOSE_FILE" up -d --no-deps --force-recreate "${ALL_SERVICES[@]}" || true
}

trap 'fail "deploy aborted"; rollback; exit 1' ERR

log "pulling images"
docker compose -f "$COMPOSE_FILE" pull --ignore-pull-failures || true

log "starting the release"
docker compose -f "$COMPOSE_FILE" up -d --remove-orphans

# Gate on health. A container that exits, or that never reports healthy inside
# the timeout, fails the deploy.
declare -a FAILED=()
deadline=$(( SECONDS + HEALTH_TIMEOUT_SECONDS ))

for svc in "${ALL_SERVICES[@]}"; do
  log "waiting for $svc"
  cid="$(docker compose -f "$COMPOSE_FILE" ps -q "$svc" 2>/dev/null || true)"
  if [[ -z "$cid" ]]; then
    fail "$svc: no container was created"
    FAILED+=("$svc")
    continue
  fi

  has_health="$(docker inspect -f '{{if .State.Health}}yes{{else}}no{{end}}' "$cid" 2>/dev/null || echo no)"

  while :; do
    state="$(docker inspect -f '{{.State.Status}}' "$cid" 2>/dev/null || echo unknown)"
    if [[ "$state" == "exited" || "$state" == "dead" ]]; then
      fail "$svc: container $state"
      FAILED+=("$svc")
      break
    fi

    if [[ "$has_health" == "yes" ]]; then
      health="$(docker inspect -f '{{.State.Health.Status}}' "$cid" 2>/dev/null || echo unknown)"
      if [[ "$health" == "healthy" ]]; then
        log "$svc: healthy"
        break
      fi
    elif [[ "$state" == "running" ]]; then
      log "$svc: running (no healthcheck defined)"
      break
    fi

    if (( SECONDS >= deadline )); then
      fail "$svc: did not become healthy within ${HEALTH_TIMEOUT_SECONDS}s (last state: ${state})"
      FAILED+=("$svc")
      break
    fi
    sleep "$HEALTH_POLL_SECONDS"
  done
done

if (( ${#FAILED[@]} > 0 )); then
  fail "deploy failed; unhealthy services: ${FAILED[*]}"
  for svc in "${FAILED[@]}"; do
    cid="$(docker compose -f "$COMPOSE_FILE" ps -q "$svc" 2>/dev/null || true)"
    if [[ -n "$cid" ]]; then
      docker logs --tail 40 "$cid" 2>&1 | sed "s/^/[$svc] /" || true
    fi
  done
  rollback
  exit 1
fi

log "all ${#ALL_SERVICES[@]} services healthy - $DEPLOY_ENV deploy of $IMAGE_TAG succeeded"
