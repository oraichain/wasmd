#!/usr/bin/env bash
# Deploy a small CosmWasm contract (the classic "hackatom" escrow) on the current
# binary, BEFORE the v0.50.15 upgrade — so 04-gov-upgrade.sh can prove the
# contract still instantiates/queries/executes correctly *after* the upgrade
# (the security patch touches x/wasm/keeper instantiate/execute/query paths and
# bumps the wasmvm version, so this is the direct regression check for that).
#
# hackatom instantiate: {"verifier": <addr>, "beneficiary": <addr>}
#   query:   {"verifier": {}}                   -> {"verifier": "<addr>"}
#   execute: {"release": {}}  (verifier only)   -> pays contract balance to beneficiary
set -euo pipefail
source "$(dirname "$0")/lib.sh"

WASM_SRC="${WASM_SRC:-${REPO_ROOT}/tests/interchaintest/bytecode/hackatom.wasm}"
[[ -f "${WASM_SRC}" ]] || die "hackatom wasm not found: ${WASM_SRC}"

running_img=$(compose ps --format '{{.Image}}' node1 2>/dev/null | head -1 || true)
[[ -n "${running_img}" ]] || die "node1 not running — run 01-init.sh .. 03-pre-upgrade.sh first"

VERIFIER_ADDR="$(cat "${DATA_DIR}/node1.addr")"
BENEFICIARY_ADDR="$(cat "${DATA_DIR}/node2.addr")"
FUND_AMOUNT="${FUND_AMOUNT:-5000000}"
GAS=(--gas auto --gas-adjustment 1.6 --gas-prices "0.0025${DENOM}" --keyring-backend test --chain-id "${CHAIN_ID}" -y -o json)

log "copy hackatom.wasm into node1 (${running_img})"
docker cp "${WASM_SRC}" tests-0.50.15-node1:/tmp/hackatom.wasm

broadcast_ok() {  # $1=label $2=json -> echoes txhash, dies on non-zero code
  local label="$1" json="$2" code hash
  code=$(echo "${json}" | grep -v 'gas estimate' | jq -r '.code // "?"')
  hash=$(echo "${json}" | grep -v 'gas estimate' | jq -r '.txhash // empty')
  [[ "${code}" == "0" ]] || die "${label} failed: code=${code} $(echo "${json}" | tail -c 300)"
  echo "${hash}"
}

wait_tx() {  # $1=hash -> echoes tx json once included
  local hash="$1" out
  for _ in $(seq 1 40); do
    out=$(compose exec -T node1 oraid query tx "${hash}" --node tcp://127.0.0.1:26657 -o json 2>/dev/null || true)
    if [[ -n "${out}" ]] && echo "${out}" | jq -e '.height and .height != "0"' >/dev/null 2>&1; then
      echo "${out}"; return 0
    fi
    sleep 1
  done
  die "timed out waiting for tx ${hash}"
}

log "store hackatom code (from node1)"
STORE_OUT=$(compose exec -T node1 oraid tx wasm store /tmp/hackatom.wasm --from node1 --node tcp://127.0.0.1:26657 "${GAS[@]}" 2>&1)
STORE_HASH=$(broadcast_ok "store" "${STORE_OUT}")
STORE_RES=$(wait_tx "${STORE_HASH}")
CODE_ID=$(echo "${STORE_RES}" | jq -r '.events[] | select(.type=="store_code") | .attributes[] | select(.key=="code_id") | .value' | head -1)
[[ -n "${CODE_ID}" && "${CODE_ID}" != "null" ]] || die "could not parse code_id from store tx"
ok "code_id=${CODE_ID}"

INIT_MSG=$(jq -nc --arg v "${VERIFIER_ADDR}" --arg b "${BENEFICIARY_ADDR}" '{verifier:$v, beneficiary:$b}')
log "instantiate hackatom (verifier=node1 ${VERIFIER_ADDR}, beneficiary=node2 ${BENEFICIARY_ADDR}, fund=${FUND_AMOUNT}${DENOM})"
INIT_OUT=$(compose exec -T node1 oraid tx wasm instantiate "${CODE_ID}" "${INIT_MSG}" \
  --from node1 --label "hackatom-upgrade-test" --admin "${VERIFIER_ADDR}" \
  --amount "${FUND_AMOUNT}${DENOM}" --node tcp://127.0.0.1:26657 "${GAS[@]}" 2>&1)
INIT_HASH=$(broadcast_ok "instantiate" "${INIT_OUT}")
INIT_RES=$(wait_tx "${INIT_HASH}")
CONTRACT_ADDR=$(echo "${INIT_RES}" | jq -r '
  [.events[] | select(.type=="instantiate") | .attributes[] | select(.key=="_contract_address") | .value] | first
')
[[ -n "${CONTRACT_ADDR}" && "${CONTRACT_ADDR}" != "null" ]] || die "could not parse contract address from instantiate tx"
ok "contract=${CONTRACT_ADDR}"

log "sanity query {\"verifier\":{}} right after deploy"
Q1=$(compose exec -T node1 oraid query wasm contract-state smart "${CONTRACT_ADDR}" '{"verifier":{}}' \
  --node tcp://127.0.0.1:26657 -o json 2>&1)
Q1_VERIFIER=$(echo "${Q1}" | jq -r '.data.verifier // .verifier // empty')
[[ "${Q1_VERIFIER}" == "${VERIFIER_ADDR}" ]] || die "pre-upgrade query mismatch: got '${Q1_VERIFIER}', want '${VERIFIER_ADDR}'"
ok "pre-upgrade query OK: verifier=${Q1_VERIFIER}"

BENEFICIARY_BAL_BEFORE=$(compose exec -T node1 oraid query bank balance "${BENEFICIARY_ADDR}" "${DENOM}" \
  --node tcp://127.0.0.1:26657 -o json 2>&1 | jq -r '.balance.amount // "0"')

cat > "${DATA_DIR}/contract.env" <<EOF
CONTRACT_ADDR=${CONTRACT_ADDR}
CODE_ID=${CODE_ID}
VERIFIER_ADDR=${VERIFIER_ADDR}
BENEFICIARY_ADDR=${BENEFICIARY_ADDR}
FUND_AMOUNT=${FUND_AMOUNT}
BENEFICIARY_BAL_BEFORE=${BENEFICIARY_BAL_BEFORE}
EOF

ok "wrote ${DATA_DIR}/contract.env — 04-gov-upgrade.sh will re-check this contract after the upgrade"
