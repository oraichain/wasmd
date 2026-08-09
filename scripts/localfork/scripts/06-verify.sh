#!/usr/bin/env bash
# Verify localfork plumbing after hard-fork height.
# NOTE: RunForkLogic is currently a noop — state-marker / apphash checks are skipped
# until production fork logic is implemented.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
# shellcheck disable=SC1091
source "${ROOT_DIR}/data/meta.env"

RPC_A="${RPC_A:-http://127.0.0.1:26657}"
RPC_S="${RPC_S:-http://127.0.0.1:26677}"

echo "==> Wait until A passes fork height ${FORK_HEIGHT}"
for i in $(seq 1 60); do
  HA=$(curl -sf "${RPC_A}/status" | jq -r '.result.sync_info.latest_block_height // 0')
  echo "  A height=${HA}"
  if [[ "${HA}" -gt "${FORK_HEIGHT}" ]]; then
    break
  fi
  sleep 2
done

HA=$(curl -sf "${RPC_A}/status" | jq -r '.result.sync_info.latest_block_height')
HS=$(curl -sf "${RPC_S}/status" | jq -r '.result.sync_info.latest_block_height')
echo "A=${HA} S=${HS}"

if [[ "${HA}" -le "${FORK_HEIGHT}" || "${HS}" -le "${FORK_HEIGHT}" ]]; then
  echo "ERROR: A/S did not pass fork height"
  exit 1
fi

echo "✓ A/S progressed past fork height ${FORK_HEIGHT}"
echo
echo "NOTE: fork.go is currently noop — re-enable mint/burn (or real migrations) checks"
echo "      in this script after production RunForkLogic is implemented."
echo "==> Summary"
echo "  A/S: past ${FORK_HEIGHT} OK"
echo "  B apphash / state marker: deferred until real fork logic lands"
