#!/bin/sh
set -e

# Print environment for debugging (remove sensitive info in production)
echo "Database configuration:"
echo "DB_HOST: $DB_HOST"
echo "DB_PORT: $DB_PORT"
echo "DB_NAME: $DB_NAME"
echo "DB_USER: $DB_USER"

# Construct the database URL from environment variables
DB_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"

echo "Waiting for PostgreSQL to be ready..."
# Simple wait-for-postgres logic
for i in $(seq 1 30); do
  pg_isready -h $DB_HOST -p $DB_PORT -U $DB_USER && break
  echo "Waiting for PostgreSQL ($i/30)..."
  sleep 1
done

echo "Applying migrations manually..."
# Apply migrations manually using psql
for migration in /root/migrations/*.sql; do
  echo "Applying migration: $migration"
  PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f $migration || echo "Warning: Migration may have already been applied: $migration"
done

echo "Migrations complete."

# Now, execute the original command (passed as arguments to the entrypoint)
echo "Starting application..."
exec "$@"