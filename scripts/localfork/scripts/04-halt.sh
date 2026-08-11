#!/usr/bin/env bash
# Finish halt: ensure S is stopped, confirm height frozen, stop A.
# Must finish with committed height < ForkHeight so A/S new binary can run fork.
# Prefer stopping S before calling this (see run-all.sh) so snapshot can use A.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "${ROOT_DIR}"

# shellcheck disable=SC1091
source "${ROOT_DIR}/data/meta.env"

if docker compose ps --status running --services 2>/dev/null | grep -qx node-s; then
  echo "==> Stopping node-s (40%) to halt consensus"
  docker compose stop node-s
else
  echo "==> node-s already stopped"
fi

echo "==> Confirming halt (height should stop increasing)"
H1=$(curl -sf http://127.0.0.1:26657/status | jq -r '.result.sync_info.latest_block_height')
sleep 4
H2=$(curl -sf http://127.0.0.1:26657/status | jq -r '.result.sync_info.latest_block_height')
echo "  height before wait=${H1} after=${H2}"
if [[ "${H1}" != "${H2}" ]]; then
  echo "WARN: height still moving; wait longer or check peers"
fi

echo "==> Stopping node-a"
docker compose stop node-a

if [[ "${H2}" -ge "${FORK_HEIGHT}" ]]; then
  echo "ERROR: halted at height=${H2} >= ForkHeight=${FORK_HEIGHT}; fork already missed on old binary"
  echo "  Re-init with higher FORK_HEIGHT / larger HALT_BUFFER"
  exit 1
fi

echo "✓ Halt complete at height=${H2} (< fork ${FORK_HEIGHT}). node-b remains on old binary."
docker compose ps
