#!/usr/bin/env bash
# Wait for the --halt-height stop to take effect on all three nodes, then confirm the chain
# is frozen one block BEFORE the fork — the state the real mainnet is sitting in today.
set -euo pipefail
source "$(dirname "$0")/env.sh"
FORK_HEIGHT=$(jq -r '.fork_height' "${FIX}")
HALT_AT=$(( FORK_HEIGHT - 1 ))

echo "==> Waiting for all nodes to reach halt-height ${HALT_AT}"
for _ in $(seq 1 120); do
  ALIVE=0
  for n in "${NODES[@]}"; do node_alive "${n}" && ALIVE=$((ALIVE+1)); done
  [[ "${ALIVE}" -eq 0 ]] && break
  sleep 2
done

# halt-height stops block production but may leave the process up; stop them for real.
LAST=""
for n in "${NODES[@]}"; do
  H=$(height "${n}")
  [[ -n "${H}" && "${H}" != "0" ]] && LAST="${H}"
  stop_node "${n}"
done

# Authoritative height comes from the committed state, not a live RPC.
H=$("${BIN_OLD}" status --home "$(node_home a)" 2>/dev/null | json | jq -r '.sync_info.latest_block_height // empty' || true)
[[ -z "${H}" ]] && H="${LAST}"
echo "  last committed height: ${H:-unknown}"

if [[ -n "${H}" && "${H}" -ge "${FORK_HEIGHT}" ]]; then
  echo "ERROR: halted at ${H} >= ForkHeight ${FORK_HEIGHT}; the old binary already passed the fork"
  exit 1
fi
echo "✓ halted below the fork (fork at ${FORK_HEIGHT}); node-b will stay on the old binary"
