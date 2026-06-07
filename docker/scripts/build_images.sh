#!/bin/sh
set -e

echo "Building AP1 Docker images..."
docker compose -f ../docker-compose.yml build

echo "Build completed. Use docker compose -f ../docker-compose.yml up"