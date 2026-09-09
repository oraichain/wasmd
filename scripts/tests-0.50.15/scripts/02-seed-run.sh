#!/usr/bin/env bash
# Run the 3-validator network LIVE on the v0.50.13b binary (full EVM era).
#
# Not just an InitChain: the chain produces a real run of blocks on v0.50.13b and
# is LEFT RUNNING, so 03 performs a genuine hot binary swap (stop live chain ->
# start v0.50.14 on the same state) rather than starting from a cold InitChain.
#
# Asserts, on v0.50.13b:
#   - all 3 validators advance in lock-step
#   - `query gov proposal ${PROPOSAL_ID}` SUCCEEDS (EVM types registered)
#   - the evm KV store is live
set -euo pipefail
source "$(dirname "$0")/lib.sh"

[[ -f "${DATA_DIR}/node1/config/genesis.json" ]] || die "run 01-init.sh first"
BLOCKS_ON_SEED="${BLOCKS_ON_SEED:-12}"

log "start node1/2/3 LIVE on ${IMAGE_SEED} (v0.50.13b, full EVM)"
pin_all_nodes "${IMAGE_SEED}"
export NODE_IMAGE="${IMAGE_SEED}"
compose up -d --force-recreate

wait_height node1 3 120 || { compose logs --tail 80 node1 || true; die "v0.50.13b network failed to start"; }
wait_height node2 3 60 || warn "node2 lagging"
wait_height node3 3 60 || warn "node3 lagging"

log "let v0.50.13b run for ~${BLOCKS_ON_SEED} blocks"
wait_height node1 "${BLOCKS_ON_SEED}" 120 || warn "did not reach ${BLOCKS_ON_SEED} (continuing)"

FAILED=0
H1=$(height_of node1); H2=$(height_of node2); H3=$(height_of node3)
log "heights: node1=${H1} node2=${H2} node3=${H3}"
[[ -n "${H1}" && -n "${H2}" && -n "${H3}" ]] && (( H1 > 3 && H2 > 3 && H3 > 3 )) \
  && ok "all 3 validators advancing on v0.50.13b" \
  || { warn "validators not all advancing"; FAILED=1; }

query_proposal node1
if [[ ${Q_RC} -eq 0 ]] && echo "${Q_OUT}" | grep -q '/cosmos.evm.vm.v1.MsgUpdateParams'; then
  ok "v0.50.13b: query gov proposal ${PROPOSAL_ID} SUCCEEDS (EVM types present)"
else
  warn "v0.50.13b: query gov proposal ${PROPOSAL_ID} failed (rc=${Q_RC}) — unexpected"
  FAILED=1
fi

EVM_STORE=$(curl -sf 'http://127.0.0.1:26657/abci_query?path=%22/store/evm/key%22&data=0x' | jq -r '.result.response.log // "?"')
if [[ "${EVM_STORE}" != *"no such store"* ]]; then
  ok "v0.50.13b: evm KV store is mounted (abci: ${EVM_STORE})"
else
  warn "v0.50.13b: evm store missing?! (${EVM_STORE})"
  FAILED=1
fi

echo
ok "network LEFT RUNNING on ${IMAGE_SEED} (node1 height $(height_of node1)) — next: ./scripts/03-pre-upgrade.sh"
exit ${FAILED}
