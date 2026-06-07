#!/bin/sh
set -e

echo "Starting AP1 containers..."
docker compose -f ../docker-compose.yml up --build
