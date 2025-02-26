#!/bin/sh
set -e

# Construct the database URL from environment variables
DB_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"

# Run migrations.  VERY IMPORTANT: Use the correct path to the migrations directory.
echo "Running migrations..."
migrate -path /root/migrations -database "${DB_URL}" up

# Check for migration errors
if [ $? -ne 0 ]; then
  echo "Migrations failed!"
  exit 1
fi

echo "Migrations complete."

# Now, execute the original command (passed as arguments to the entrypoint)
exec "$@"