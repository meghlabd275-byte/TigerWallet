#!/bin/bash
# Creates the per-service databases that this compose stack expects.
#
# The official postgres image only reads POSTGRES_DB, so a
# POSTGRES_MULTIPLE_DATABASES variable is silently ignored and every service
# except the one whose name matches POSTGRES_DB fails to connect at boot. This
# script runs once on first initialisation (docker-entrypoint-initdb.d) and
# creates the remaining databases, each owned by the configured superuser so the
# service credentials already in the compose file work unchanged.
set -euo pipefail

: "${POSTGRES_USER:?POSTGRES_USER must be set}"
: "${POSTGRES_DB:?POSTGRES_DB must be set}"

# The databases the services connect to. Keep in step with the DB_NAME values in
# docker-compose.yml; POSTGRES_DB is included so a stack where it differs from
# the primary name still gets it.
databases=(
  "${POSTGRES_DB}"
  tigerwallet
  tigerwallet_listing
  tigerwallet_wallet
  tigerwallet_swap
  tigerwallet_bots
  tigerwallet_nft
)

for db in "${databases[@]}"; do
  # Identifiers cannot be parameterised in CREATE DATABASE, so the name is
  # restricted to the character set these fixed names use rather than being
  # interpolated unchecked.
  if [[ ! "${db}" =~ ^[a-z_][a-z0-9_]*$ ]]; then
    echo "init-databases: refusing unsafe database name: ${db}" >&2
    exit 1
  fi

  exists=$(psql --username "${POSTGRES_USER}" --dbname postgres --tuples-only --no-align \
    --command "SELECT 1 FROM pg_database WHERE datname = '${db}'")
  if [ "${exists}" = "1" ]; then
    echo "init-databases: ${db} already exists"
    continue
  fi

  echo "init-databases: creating ${db}"
  psql --username "${POSTGRES_USER}" --dbname postgres \
    --command "CREATE DATABASE \"${db}\" OWNER \"${POSTGRES_USER}\""
done

echo "init-databases: done"
