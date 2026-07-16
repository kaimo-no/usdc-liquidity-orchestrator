#!/usr/bin/env bash
# Dry plan against a local orchestrator (go run ./cmd/server).
# Default: Arc Testnet + circle_gateway (hackathon example).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ADDR="${LISTEN_ADDR:-http://127.0.0.1:8088}"
EXAMPLE="${1:-$ROOT/examples/plan.json}"
curl -sS -X POST "$ADDR/v1/plan" \
  -H 'Content-Type: application/json' \
  --data-binary @"$EXAMPLE" | jq .
