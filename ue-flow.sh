#!/usr/bin/env bash
set -euo pipefail

AMF_URL="${AMF_URL:-http://localhost:8001}"
UE_URL="${UE_URL:-http://localhost:8006}"
IMSI="${IMSI:-310150123456789}"

echo "[1/3] UE attach"
curl -sS -X POST "$UE_URL/attach" \
  -H 'Content-Type: application/json' \
  -d "{\"imsi\":\"$IMSI\"}"

echo

echo "[2/3] Session status"
curl -sS "$UE_URL/session-status"

echo

echo "[3/3] UE release"
curl -sS -X POST "$UE_URL/release" \
  -H 'Content-Type: application/json' \
  -d "{\"imsi\":\"$IMSI\"}"

echo
