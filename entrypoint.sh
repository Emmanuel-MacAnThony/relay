#!/bin/sh
set -e

MIGRATE_URL=$(echo "$DATABASE_URL" | sed 's/^postgres:\/\//pgx5:\/\//')

echo "Running migrations..."
migrate -path /app/db/migrations -database "$MIGRATE_URL" up
echo "Migrations complete."

echo "Starting Relay..."
exec /app/relay
