#!/usr/bin/env bash
# On-chain governance software-upgrade of the running 3-validator network to
# ${UPGRADE_NAME} (app/upgrades/v05015/upgrades.go):
#
#   1. submit MsgSoftwareUpgrade  (plan name=${UPGRADE_NAME}, height=H)
#   2. all 3 validators vote yes; wait PASSED; assert `upgrade plan` scheduled
#   3. wait until every node HALTS at H  ('UPGRADE "..." NEEDED')
#   4. swap all 3 to the NEW binary -> they apply the upgrade at H and resume
#   5. assert the v0.50.15 handler ran, `query gov proposal ${PROPOSAL_ID}` now
#      works on all 3, and the NEW oraid is a PIE
#
# Network is LEFT RUNNING on the NEW binary.
set -euo pipefail
source "$(dirname "$0")/lib.sh"

docker image inspect "${IMAGE_NEW}" >/dev/null 2>&1 || die "missing ${IMAGE_NEW}"
compose ps --status running 2>/dev/null | grep -q tests-0.50.15-node1 || die "network not running — run 03-pre-upgrade.sh first"

GAS=(--gas auto --gas-adjustment 1.5 --gas-prices "0.0025${DENOM}" --keyring-backend test --chain-id "${CHAIN_ID}" -y -o json)
NEW_PROP=$(( PROPOSAL_ID + 1 ))

SUBMIT_H=$(height_of node1)
UPGRADE_H=$(( SUBMIT_H + 60 ))
log "submit software-upgrade '${UPGRADE_NAME}' at height ${UPGRADE_H} (now ${SUBMIT_H}), proposal id ${NEW_PROP}"

set +e
SUB=$(compose exec -T node1 oraid tx upgrade software-upgrade "${UPGRADE_NAME}" \
        --upgrade-height "${UPGRADE_H}" --upgrade-info '{}' --no-validate \
        --title "upgrade ${UPGRADE_NAME}" --summary "register evmlegacy decode stub" \
        --deposit "20000000${DENOM}" --from node1 --node tcp://127.0.0.1:26657 "${GAS[@]}" 2>&1)
SUB_RC=$?
set -e
echo "${SUB}" | tail -c 400; echo
[[ ${SUB_RC} -eq 0 ]] || die "submit failed"
echo "${SUB}" | grep -v 'gas estimate' | jq -e '.code == 0' >/dev/null 2>&1 || warn "submit tx code != 0 (may still be in mempool)"

log "wait for proposal ${NEW_PROP} to enter voting"
for _ in $(seq 1 30); do
  compose exec -T node1 oraid query gov proposal "${NEW_PROP}" --node tcp://127.0.0.1:26657 -o json >/dev/null 2>&1 && break
  sleep 1
done

vote() {
  local n="$1"
  set +e
  local o; o=$(compose exec -T "${n}" oraid tx gov vote "${NEW_PROP}" yes \
    --from "${n}" --node tcp://127.0.0.1:26657 "${GAS[@]}" 2>&1)
  local rc=$?
  set -e
  echo "${o}" | grep -v 'gas estimate' | jq -e '.code == 0' >/dev/null 2>&1 && ok "${n} voted yes" || warn "${n} vote rc=${rc}: $(echo "${o}" | tail -c 200)"
  sleep 2
}
log "vote yes from all 3 validators"
vote node2; vote node3; vote node1

log "wait for proposal ${NEW_PROP} -> PASSED"
STATUS=
for _ in $(seq 1 40); do
  STATUS=$(compose exec -T node1 oraid query gov proposal "${NEW_PROP}" --node tcp://127.0.0.1:26657 -o json 2>/dev/null \
           | jq -r '.proposal.status // .status // empty')
  case "${STATUS}" in
    PROPOSAL_STATUS_PASSED|3) ok "proposal ${NEW_PROP} PASSED"; break ;;
    PROPOSAL_STATUS_REJECTED|PROPOSAL_STATUS_FAILED|4|5) die "proposal ended ${STATUS}" ;;
  esac
  sleep 1
done
[[ "${STATUS}" == "PROPOSAL_STATUS_PASSED" || "${STATUS}" == "3" ]] || die "proposal not PASSED (status=${STATUS})"

PLAN=$(compose exec -T node1 oraid query upgrade plan --node tcp://127.0.0.1:26657 -o json 2>/dev/null || true)
echo "  upgrade plan: $(echo "${PLAN}" | tr -d '\n ')"
# `oraid q upgrade plan` returns either {"plan":{...}} or the bare plan object.
if echo "${PLAN}" | jq -e --arg n "${UPGRADE_NAME}" '(.plan.name // .name) == $n' >/dev/null 2>&1; then
  ok "upgrade '${UPGRADE_NAME}' scheduled at height $(echo "${PLAN}" | jq -r '.plan.height // .height')"
else
  die "upgrade plan not scheduled"
fi

log "wait for the network to HALT at height ${UPGRADE_H}"
halted=0
for _ in $(seq 1 150); do
  if compose logs node1 2>&1 | grep -qE "UPGRADE \"${UPGRADE_NAME}\" NEEDED at height: ${UPGRADE_H}"; then
    halted=1; break
  fi
  st=$(docker inspect -f '{{.State.Status}}' tests-0.50.15-node1 2>/dev/null || echo "?")
  [[ "${st}" == "exited" ]] && { halted=1; break; }
  sleep 1
done
[[ ${halted} -eq 1 ]] || { compose logs --tail 40 node1; die "network did not halt at ${UPGRADE_H}"; }
ok "halted at ${UPGRADE_H}  ('UPGRADE \"${UPGRADE_NAME}\" NEEDED')"
compose logs node1 2>&1 | grep -E "UPGRADE \"${UPGRADE_NAME}\" NEEDED" | tail -1
sleep 3
compose stop >/dev/null 2>&1 || true

#######################################################################
# swap all 3 to the NEW binary -> apply upgrade at ${UPGRADE_H}
#######################################################################
log "swap node1/2/3 -> ${IMAGE_NEW}  (ELF Type: $(image_elf_type "${IMAGE_NEW}")) and resume"
pin_all_nodes "${IMAGE_NEW}"
export NODE_IMAGE="${IMAGE_NEW}"
compose up -d --force-recreate

wait_height node1 "$(( UPGRADE_H + 2 ))" 120 || { compose logs --tail 60 node1; die "network did not resume past ${UPGRADE_H} on ${IMAGE_NEW}"; }
wait_height node2 "$(( UPGRADE_H + 1 ))" 60 || warn "node2 lagging"
wait_height node3 "$(( UPGRADE_H + 1 ))" 60 || warn "node3 lagging"

FAILED=0
# Authoritative: x/upgrade records the height the plan was applied at.
applied_h=""
for _ in $(seq 1 20); do
  applied_h=$(compose exec -T node1 oraid query upgrade applied "${UPGRADE_NAME}" \
                --node tcp://127.0.0.1:26657 2>/dev/null | awk -F'"' '/height:/ {print $2}')
  [[ -n "${applied_h}" && "${applied_h}" != "0" ]] && break
  sleep 1
done
if [[ -n "${applied_h}" && "${applied_h}" != "0" ]]; then
  ok "upgrade '${UPGRADE_NAME}' applied at height ${applied_h} (query upgrade applied)"
  compose logs node1 2>&1 | grep -E "applying upgrade \"${UPGRADE_NAME}\"" | tail -1 || true
else
  warn "x/upgrade has no applied record for '${UPGRADE_NAME}'"
  FAILED=1
fi

log "post-upgrade: query gov proposal ${PROPOSAL_ID} on all 3"
for n in "${NODES[@]}"; do
  query_proposal "${n}"
  if [[ ${Q_RC} -eq 0 ]] && echo "${Q_OUT}" | grep -q '/cosmos.evm.vm.v1.MsgUpdateParams'; then
    ok "${n}: proposal ${PROPOSAL_ID} decodes"
  else
    warn "${n}: query still failing (rc=${Q_RC}): $(echo "${Q_OUT}" | head -c 200)"
    FAILED=1
  fi
done

ELF_NEW=$(image_elf_type "${IMAGE_NEW}")
[[ "${ELF_NEW}" == *DYN* ]] && ok "NEW oraid is PIE (${ELF_NEW}) — Makefile BUILD_MODE=pie default" \
  || { warn "NEW oraid not PIE (${ELF_NEW})"; FAILED=1; }

echo
if [[ ${FAILED} -eq 0 ]]; then
  printf '\033[1;32mPASS\033[0m  3-validator gov upgrade %s applied at height %s; proposal %s now queryable on all nodes; binary is PIE.\n' \
    "${UPGRADE_NAME}" "${UPGRADE_H}" "${PROPOSAL_ID}"
else
  printf '\033[1;31mFAIL\033[0m  see !! lines above.\n'
fi

echo
H1=$(height_of node1); H2=$(height_of node2); H3=$(height_of node3)
ok "network LEFT RUNNING on ${IMAGE_NEW}  (heights: node1=${H1:-?} node2=${H2:-?} node3=${H3:-?})"
cat <<EOF

  RPC:  node1 $(rpc_of node1)   node2 $(rpc_of node2)   node3 $(rpc_of node3)
  inspect:
    docker compose -f scripts/tests-0.50.15/docker-compose.yml logs -f node1
    docker compose -f scripts/tests-0.50.15/docker-compose.yml exec node2 \\
      oraid query gov proposal ${PROPOSAL_ID} --node tcp://127.0.0.1:26657 -o json
    docker compose -f scripts/tests-0.50.15/docker-compose.yml exec node3 \\
      oraid query upgrade applied ${UPGRADE_NAME} --node tcp://127.0.0.1:26657
  stop: ./scripts/99-clean.sh
EOF
exit ${FAILED}
