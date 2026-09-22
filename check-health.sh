#!/usr/bin/env bash
set -euo pipefail

services=(
  "http://localhost:8000/health"
  "http://localhost:8001/health"
  "http://localhost:8002/health"
  "http://localhost:8003/health"
  "http://localhost:8004/health"
  "http://localhost:8005/health"
)

for url in "${services[@]}"; do
  echo "Checking $url"
  curl -fsS "$url" || {
    echo "Health check failed for $url" >&2
    exit 1
  }
done

echo "All 5G core services are healthy."
