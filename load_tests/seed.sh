#!/usr/bin/env bash
# Seed verified load-test users directly into the database.
# Requires the postgres container to be running.
#
# Inserts:
#   loadtest@musicroom.test          (used by delegation + playlist tests)
#   loadtest1@musicroom.test .. loadtest150@musicroom.test  (one per VU for track_vote)
#
# Password for all accounts: loadtest123  (bcrypt cost 10)
#
# Usage: bash load_tests/seed.sh
set -euo pipefail

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.yml}"
PG_USER="${POSTGRES_USER:-postgres}"
PG_DB="${POSTGRES_DB:-musicroom}"

HASH='$2b$10$0P6yW1xtacTOkmI/HA72e.JK9DO8ddItqInke2B0mkpP1FYic0QgG'

docker compose -f "$COMPOSE_FILE" exec -T postgres psql -U "$PG_USER" -d "$PG_DB" -c "
  INSERT INTO users (id, email, password_hash, is_verified)
  VALUES (gen_random_uuid(), 'loadtest@musicroom.test', '${HASH}', true)
  ON CONFLICT (email) DO UPDATE SET is_verified = true;

  INSERT INTO users (id, email, password_hash, is_verified)
  SELECT
    gen_random_uuid(),
    'loadtest' || i || '@musicroom.test',
    '${HASH}',
    true
  FROM generate_series(1, 150) AS s(i)
  ON CONFLICT (email) DO UPDATE SET is_verified = true;
" 2>&1

echo ">> Done. loadtest@musicroom.test + loadtest1..150@musicroom.test / loadtest123"
