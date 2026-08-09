#!/usr/bin/env bash
# Halt chain: stop S (40%) first so A+B=60% cannot continue, then stop A.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "${ROOT_DIR}"

echo "==> Stopping node-s (40%) to halt consensus"
docker compose stop node-s

echo "==> Confirming halt (height should stop increasing)"
H1=$(curl -sf http://127.0.0.1:26657/status | jq -r '.result.sync_info.latest_block_height')
sleep 6
H2=$(curl -sf http://127.0.0.1:26657/status | jq -r '.result.sync_info.latest_block_height')
echo "  height before wait=${H1} after=${H2}"
if [[ "${H1}" != "${H2}" ]]; then
  echo "WARN: height still moving; wait longer or check peers"
fi

echo "==> Stopping node-a"
docker compose stop node-a

echo "✓ Halt complete. node-b remains on old binary (running or idle)."
docker compose ps
