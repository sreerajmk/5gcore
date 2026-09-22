#!/usr/bin/env bash
set -euo pipefail

docker-compose down -v --remove-orphans || docker compose down -v --remove-orphans

echo "5G core services stopped and cleaned up."
