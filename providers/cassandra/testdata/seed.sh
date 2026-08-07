#!/usr/bin/env bash
# Seed fixtures for the cassandra provider integration test.
#
# All credentials come from the environment so no secrets are committed:
#   CASS_HOST            host (default 127.0.0.1)
#   CASS_PORT            port (default 9042)
#   CASS_USER            superuser (default cassandra)
#   CASS_PASSWORD        superuser password (required)
#   CASS_AUDITOR_PASSWORD  password for the seeded auditor role (required)
#   CASS_CQLSH           cqlsh command (default "cqlsh"; e.g. "docker exec cass-test cqlsh")
#
# Usage: CASS_PASSWORD=... CASS_AUDITOR_PASSWORD=... ./seed.sh
set -euo pipefail

HOST="${CASS_HOST:-127.0.0.1}"
PORT="${CASS_PORT:-9042}"
USER="${CASS_USER:-cassandra}"
PASS="${CASS_PASSWORD:?set CASS_PASSWORD}"
AUDITOR_PASS="${CASS_AUDITOR_PASSWORD:?set CASS_AUDITOR_PASSWORD}"
CQLSH="${CASS_CQLSH:-cqlsh}"

# shellcheck disable=SC2086
$CQLSH -u "$USER" -p "$PASS" "$HOST" "$PORT" -e "
  -- A least-privilege auditor role to audit against the default cassandra superuser.
  CREATE ROLE IF NOT EXISTS auditor WITH LOGIN = true AND PASSWORD = '${AUDITOR_PASS}';
  -- A SimpleStrategy, replication-factor-1 keyspace to trip the replication posture check.
  CREATE KEYSPACE IF NOT EXISTS app WITH replication = {'class':'SimpleStrategy','replication_factor':1};
  -- A scoped grant to the auditor (a non-superuser with a narrow data permission).
  GRANT SELECT ON KEYSPACE app TO auditor;
"

echo "seed complete"
