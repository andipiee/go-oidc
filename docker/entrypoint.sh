#!/bin/sh
set -e

if [ "${AUTO_MIGRATE}" = "true" ]; then
  echo "Running database migrations..."
  ./migrate up
fi

echo "Starting server..."
exec ./server
