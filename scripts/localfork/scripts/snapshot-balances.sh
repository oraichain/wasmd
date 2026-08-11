#!/usr/bin/env bash
# Snapshot ORAI + CW20 + RecoveryNativeDenoms balances for localfork fork comparison.
# Usage: ./snapshot-balances.sh pre|post
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PHASE="${1:?usage: $0 pre|post}"
OUT="${ROOT_DIR}/data/balances-${PHASE}-fork.json"
RPC="${RPC_A:-http://127.0.0.1:26657}"
LCD="${LCD_A:-http://127.0.0.1:1317}"
DENOM="${DENOM:-orai}"
BLACKLIST_JSON="${BLACKLIST_JSON:-${ROOT_DIR}/blacklist-addresses.json}"
NATIVE_JSON="${NATIVE_JSON:-${ROOT_DIR}/data/native-denoms.json}"
[[ -f "${NATIVE_JSON}" ]] || NATIVE_JSON="${ROOT_DIR}/native-denoms.json"

HEIGHT=$(curl -sf "${RPC}/status" | jq -r '.result.sync_info.latest_block_height')
TESTER=""
DEPLOYER=""
[[ -f "${ROOT_DIR}/data/tester.address" ]] && TESTER=$(cat "${ROOT_DIR}/data/tester.address")
[[ -f "${ROOT_DIR}/data/cw20-deployer.address" ]] && DEPLOYER=$(cat "${ROOT_DIR}/data/cw20-deployer.address")

CW20_CONTRACT=""
CW20_CONTRACT_2=""
CW20_FROM=""
CW20_TO=""
CW20_AMOUNT=""
CW20_AMOUNT_2=""
if [[ -f "${ROOT_DIR}/data/cw20.env" ]]; then
  # shellcheck disable=SC1091
  source "${ROOT_DIR}/data/cw20.env"
fi

NATIVE_DENOMS=()
if [[ -f "${NATIVE_JSON}" ]]; then
  while IFS= read -r d; do
    [[ -n "${d}" ]] && NATIVE_DENOMS+=("${d}")
  done < <(jq -r '.[].denom' "${NATIVE_JSON}")
fi

orai_bal() {
  local addr="$1"
  curl -sf "${LCD}/cosmos/bank/v1beta1/balances/${addr}/by_denom?denom=${DENOM}" \
    | jq -r '.balance.amount // "0"' 2>/dev/null || echo "0"
}

native_bal() {
  local addr="$1"
  local denom="$2"
  curl -sf "${LCD}/cosmos/bank/v1beta1/balances/${addr}/by_denom?denom=${denom}" \
    | jq -r '.balance.amount // "0"' 2>/dev/null || echo "0"
}

cw20_bal() {
  local contract="$1"
  local addr="$2"
  if [[ -z "${contract}" || -z "${addr}" ]]; then
    echo "0"
    return
  fi
  local q
  q=$(jq -nc --arg a "${addr}" '{balance:{address:$a}}')
  docker exec -e HOME=/orai localfork-a oraid query wasm contract-state smart "${contract}" "${q}" \
    --home /orai/.oraid --node tcp://127.0.0.1:26657 --output json 2>/dev/null \
    | jq -r '.data.balance // .balance // "0"' || echo "0"
}

native_map_for() {
  local addr="$1"
  local obj='{}'
  local d bal
  for d in "${NATIVE_DENOMS[@]+"${NATIVE_DENOMS[@]}"}"; do
    bal=$(native_bal "${addr}" "${d}")
    obj=$(jq -nc --argjson o "${obj}" --arg d "${d}" --arg a "${bal}" '$o + {($d):$a}')
  done
  echo "${obj}"
}

ROWS='[]'
add_row() {
  local role="$1" addr="$2" orai="$3" cw20_a="$4" cw20_b="$5" natives="$6"
  ROWS=$(jq -nc --argjson rows "${ROWS}" \
    --arg role "${role}" --arg addr "${addr}" \
    --arg orai "${orai}" --arg cw20_a "${cw20_a}" --arg cw20_b "${cw20_b}" \
    --argjson natives "${natives}" \
    '$rows + [{role:$role,address:$addr,orai:$orai,cw20_v1:$cw20_a,cw20_v2:$cw20_b,native:$natives}]')
}

while IFS= read -r addr; do
  [[ -z "${addr}" ]] && continue
  role="blacklist"
  if [[ "${addr}" == "${CW20_FROM:-}" ]]; then
    role="blacklist+recovery_from"
  fi
  add_row "${role}" "${addr}" "$(orai_bal "${addr}")" \
    "$(cw20_bal "${CW20_CONTRACT}" "${addr}")" \
    "$(cw20_bal "${CW20_CONTRACT_2}" "${addr}")" \
    "$(native_map_for "${addr}")"
done < <(jq -r '.[]' "${BLACKLIST_JSON}")

REVERT_JSON="${REVERT_JSON:-${ROOT_DIR}/revert-addresses.json}"
if [[ -f "${REVERT_JSON}" ]]; then
  while IFS=$'\t' read -r addr keep to_rec; do
    [[ -z "${addr}" ]] && continue
    role="revert(keep=${keep})"
    if [[ "${to_rec}" == "true" ]]; then
      role="revert(pool→recovery,burn=${keep})"
    fi
    add_row "${role}" "${addr}" "$(orai_bal "${addr}")" \
      "$(cw20_bal "${CW20_CONTRACT}" "${addr}")" \
      "$(cw20_bal "${CW20_CONTRACT_2}" "${addr}")" \
      "$(native_map_for "${addr}")"
  done < <(jq -r '.[] | [.address, .amount, (.send_remainder_to_recovery // false)] | @tsv' "${REVERT_JSON}")
fi

if [[ -n "${TESTER}" ]]; then
  add_row "tester/recovery_to" "${TESTER}" "$(orai_bal "${TESTER}")" \
    "$(cw20_bal "${CW20_CONTRACT}" "${TESTER}")" \
    "$(cw20_bal "${CW20_CONTRACT_2}" "${TESTER}")" \
    "$(native_map_for "${TESTER}")"
fi

if [[ -n "${DEPLOYER}" ]]; then
  add_row "cw20_deployer" "${DEPLOYER}" "$(orai_bal "${DEPLOYER}")" \
    "$(cw20_bal "${CW20_CONTRACT}" "${DEPLOYER}")" \
    "$(cw20_bal "${CW20_CONTRACT_2}" "${DEPLOYER}")" \
    "$(native_map_for "${DEPLOYER}")"
fi

NATIVE_META='[]'
if [[ -f "${NATIVE_JSON}" ]]; then
  NATIVE_META=$(jq -c '.' "${NATIVE_JSON}")
fi

jq -nc \
  --arg phase "${PHASE}" \
  --arg height "${HEIGHT}" \
  --arg denom "${DENOM}" \
  --arg cw20_v1 "${CW20_CONTRACT}" \
  --arg cw20_v2 "${CW20_CONTRACT_2}" \
  --arg cw20_amount "${CW20_AMOUNT}" \
  --arg cw20_amount_2 "${CW20_AMOUNT_2}" \
  --argjson native_denoms "${NATIVE_META}" \
  --argjson wallets "${ROWS}" \
  '{phase:$phase,height:($height|tonumber),denom:$denom,cw20_contract_v1:$cw20_v1,cw20_contract_v2:$cw20_v2,cw20_mint_amount:$cw20_amount,cw20_mint_amount_2:$cw20_amount_2,native_denoms:$native_denoms,wallets:$wallets}' \
  > "${OUT}"

echo "✓ wrote ${OUT} (height=${HEIGHT}, wallets=$(echo "${ROWS}" | jq 'length'), native_denoms=${#NATIVE_DENOMS[@]})"
