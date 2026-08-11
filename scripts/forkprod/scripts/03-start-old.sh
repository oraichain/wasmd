#!/usr/bin/env bash
# Start all three validators on the OLD (v0.50.13b) binary, with --halt-height one block
# before the fork. This reproduces the real mainnet halt: every node stops cleanly at a
# known height instead of racing a stopwatch.
set -euo pipefail
source "$(dirname "$0")/env.sh"
FORK_HEIGHT=$(jq -r '.fork_height' "${FIX}")
HALT_AT=$(( FORK_HEIGHT - 1 ))

stop_all
echo "==> Starting 3 validators on the old binary (halt-height ${HALT_AT}, fork at ${FORK_HEIGHT})"
for n in "${NODES[@]}"; do
  : > "$(node_home "${n}")/start.log"
  start_node "${n}" "${BIN_OLD}" --halt-height "${HALT_AT}"
  echo "  node-${n} (rpc $(port_rpc "${n}"))"
done

for _ in $(seq 1 60); do
  sleep 2
  H=$(height a)
  [[ "${H}" != "0" && "${H}" != "null" && -n "${H}" ]] && { echo "  A height=${H}"; break; }
done
if [[ "$(height a)" == "0" || -z "$(height a)" ]]; then
  echo "ERROR: chain did not start"; tail -25 "$(node_home a)/start.log"; exit 1
fi
echo "✓ 3 validators up on the old binary"
