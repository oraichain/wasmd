#!/usr/bin/env bash
# Genuine hot binary swap: v0.50.13b (live) -> v0.50.14, no gov proposal.
#
#   1. capture the live chain's height H and the app hash CometBFT recorded for H
#   2. stop the running v0.50.13b nodes
#   3. start v0.50.14 on the SAME data dirs
#   4. assert v0.50.14:
#        - resumes and advances past H          (state machine accepted v0.50.13b state)
#        - reports the SAME app hash for H       (deterministic continuity across the swap)
#        - `query gov proposal ${PROPOSAL_ID}` now FAILS   (EVM types no longer registered)
#        - still mounts the evm KV store         (soft-remove: module gone, store kept)
#
# Nodes are LEFT RUNNING for 04-gov-upgrade.sh.
set -euo pipefail
source "$(dirname "$0")/lib.sh"

docker image inspect "${IMAGE_OLD}" >/dev/null 2>&1 || die "missing ${IMAGE_OLD} — run 00-build.sh"

running_img=$(compose ps --format '{{.Image}}' node1 2>/dev/null | head -1 || true)
[[ "${running_img}" == "${IMAGE_SEED}" ]] \
  || die "expected node1 running on ${IMAGE_SEED}, got '${running_img}' — run 02-seed-run.sh first"

app_hash_at() {  # app_hash CometBFT recorded in the header at height $1
  curl -sf "$(rpc_of node1)/block?height=$1" 2>/dev/null \
    | jq -r '.result.block.header.app_hash // empty'
}

H=$(height_of node1)
[[ -n "${H}" ]] || die "cannot read live height"
AH_BEFORE=$(app_hash_at "${H}")
log "pre-swap: v0.50.13b live at height ${H}, app_hash(${H})=${AH_BEFORE:-?}"

log "stop v0.50.13b nodes"
compose stop

log "start node1/2/3 on ${IMAGE_OLD} (v0.50.14, ELF $(image_elf_type "${IMAGE_OLD}")) — same state"
pin_all_nodes "${IMAGE_OLD}"
export NODE_IMAGE="${IMAGE_OLD}"
compose up -d --force-recreate

FAILED=0
if wait_height node1 "$(( H + 2 ))" 90; then
  ok "v0.50.14 resumed and advanced past ${H} (accepted v0.50.13b state)"
else
  compose logs --tail 60 node1 || true
  die "v0.50.14 did NOT resume from v0.50.13b state"
fi
wait_height node2 "$(( H + 1 ))" 60 || warn "node2 lagging"
wait_height node3 "$(( H + 1 ))" 60 || warn "node3 lagging"

AH_AFTER=$(app_hash_at "${H}")
if [[ -n "${AH_BEFORE}" && "${AH_AFTER}" == "${AH_BEFORE}" ]]; then
  ok "app_hash(${H}) identical across the swap (${AH_AFTER}) — deterministic continuity"
else
  warn "app_hash(${H}) mismatch: before=${AH_BEFORE:-?} after=${AH_AFTER:-?}"
  FAILED=1
fi

for n in "${NODES[@]}"; do
  query_proposal "${n}"
  if [[ ${Q_RC} -ne 0 ]] && echo "${Q_OUT}" | grep -qiE 'resolve|no concrete type|MsgUpdateParams'; then
    ok "${n}: query gov proposal ${PROPOSAL_ID} now FAILS (EVM removed in v0.50.14)"
  else
    warn "${n}: query did not fail as expected (rc=${Q_RC}): $(echo "${Q_OUT}" | head -c 160)"
    FAILED=1
  fi
done

EVM_STORE=$(curl -sf 'http://127.0.0.1:26657/abci_query?path=%22/store/evm/key%22&data=0x' | jq -r '.result.response.log // "?"')
if [[ "${EVM_STORE}" != *"no such store"* ]]; then
  ok "v0.50.14 still mounts the evm KV store (soft-remove: ${EVM_STORE})"
else
  warn "v0.50.14 unexpectedly dropped the evm store (${EVM_STORE})"
  FAILED=1
fi

echo
[[ ${FAILED} -eq 0 ]] \
  && printf '\033[1;32mPASS\033[0m  v0.50.13b -> v0.50.14 hot swap: state carried over, EVM queries now broken.\n' \
  || printf '\033[1;31mFAIL\033[0m  see !! lines above.\n'
ok "network LEFT RUNNING on ${IMAGE_OLD} — next: ./scripts/04-gov-upgrade.sh"
exit ${FAILED}
