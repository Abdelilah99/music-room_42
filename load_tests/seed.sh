#!/usr/bin/env bash
# Seed a verified load-test user directly into the database.
# Requires the postgres container to be running.
#
# Usage: bash load_tests/seed.sh
#
# Password: loadtest123  (bcrypt cost 10)
set -euo pipefail

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.yml}"
PG_USER="${POSTGRES_USER:-postgres}"
PG_DB="${POSTGRES_DB:-musicroom}"

HASH='$2b$10$0P6yW1xtacTOkmI/HA72e.JK9DO8ddItqInke2B0mkpP1FYic0QgG'

echo ">> Seeding load-test user..."
docker compose -f "$COMPOSE_FILE" exec -T postgres psql -U "$PG_USER" -d "$PG_DB" -c "
  INSERT INTO users (id, email, password_hash, is_verified)
  VALUES (
    gen_random_uuid(),
    'loadtest@musicroom.test',
    '${HASH}',
    true
  )
  ON CONFLICT (email) DO UPDATE SET is_verified = true;
" 2>&1

echo ">> Done. Test credentials: loadtest@musicroom.test / loadtest123"
