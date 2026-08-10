#!/usr/bin/env bash
# Verify hard-fork burn + blacklist after ForkHeight, and node-b apphash divergence.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
# shellcheck disable=SC1091
source "${ROOT_DIR}/data/meta.env"

RPC_A="${RPC_A:-http://127.0.0.1:26657}"
RPC_S="${RPC_S:-http://127.0.0.1:26677}"
LCD_A="${LCD_A:-http://127.0.0.1:1317}"
BLACKLIST_JSON="${BLACKLIST_JSON:-${ROOT_DIR}/blacklist-addresses.json}"

echo "==> Wait until A passes fork height ${FORK_HEIGHT}"
for i in $(seq 1 90); do
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

echo "==> Check blacklist addresses burned on A (balance orai == 0)"
ADDRS=()
while IFS= read -r line; do
  [[ -n "${line}" ]] && ADDRS+=("${line}")
done < <(jq -r '.[]' "${BLACKLIST_JSON}")
if [[ "${#ADDRS[@]}" -eq 0 ]]; then
  echo "ERROR: empty blacklist at ${BLACKLIST_JSON}"
  exit 1
fi

FAIL=0
for addr in "${ADDRS[@]}"; do
  BAL=$(curl -sf "${LCD_A}/cosmos/bank/v1beta1/balances/${addr}/by_denom?denom=${DENOM}" \
    | jq -r '.balance.amount // "0"')
  echo "  ${addr} bal=${BAL}"
  if [[ "${BAL}" != "0" ]]; then
    echo "ERROR: expected burned (0), got ${BAL}"
    FAIL=1
  fi
done
if [[ "${FAIL}" -ne 0 ]]; then
  exit 1
fi
echo "✓ All ${#ADDRS[@]} blacklist balances burned on A"

REVERT_JSON="${REVERT_JSON:-${ROOT_DIR}/revert-addresses.json}"
POOL_REMAINDER_TO_RECOVERY=0
if [[ -f "${REVERT_JSON}" ]]; then
  echo "==> Check revert addresses on A"
  REV_FAIL=0
  while IFS=$'\t' read -r REV_ADDR REV_AMT REV_GEN REV_TO_REC; do
    [[ -z "${REV_ADDR}" ]] && continue
    BAL=$(curl -sf "${LCD_A}/cosmos/bank/v1beta1/balances/${REV_ADDR}/by_denom?denom=${DENOM}" \
      | jq -r '.balance.amount // "0"')
    if [[ "${REV_TO_REC}" == "true" ]]; then
      # Special-case pool: burn Amount then send remainder → RecoveryAddress (pool bal == 0).
      echo "  ${REV_ADDR} bal=${BAL} (pool remainder→recovery; want 0; burn=${REV_AMT} genesis=${REV_GEN})"
      if [[ "${BAL}" != "0" ]]; then
        echo "ERROR: pool revert address expected 0 after remainder send, got ${BAL}"
        REV_FAIL=1
      fi
      POOL_REMAINDER_TO_RECOVERY=$((POOL_REMAINDER_TO_RECOVERY + REV_GEN - REV_AMT))
    else
      echo "  ${REV_ADDR} bal=${BAL} (want keep=${REV_AMT})"
      if [[ "${BAL}" != "${REV_AMT}" ]]; then
        echo "ERROR: revert address expected ${REV_AMT}, got ${BAL}"
        REV_FAIL=1
      fi
    fi
  done < <(jq -r '.[] | [.address, .amount, .genesis, (.send_remainder_to_recovery // false)] | @tsv' "${REVERT_JSON}")
  if [[ "${REV_FAIL}" -ne 0 ]]; then
    exit 1
  fi
  echo "✓ All revert addresses OK (pool remainder→recovery=${POOL_REMAINDER_TO_RECOVERY}${DENOM})"
fi

# ---------------------------------------------------------------------------
# RecoveryAddress balances BEFORE bank-send tests (true post-fork state).
# ---------------------------------------------------------------------------
echo "==> RecoveryAddress balances: pre-fork snapshot vs post-fork (before bank sends)"
CW20_ENV="${ROOT_DIR}/data/cw20.env"
if [[ ! -f "${CW20_ENV}" ]]; then
  echo "ERROR: missing ${CW20_ENV} (run 02b-deploy-cw20.sh)"
  exit 1
fi
# shellcheck disable=SC1090
source "${CW20_ENV}"

TESTER_KEY="${TESTER_KEY:-tester}"
TESTER_ADDR="${TESTER_ADDR:-$(cat "${ROOT_DIR}/data/tester.address" 2>/dev/null || true)}"
if [[ -z "${TESTER_ADDR}" ]]; then
  TESTER_ADDR=$(jq -r .address "${ROOT_DIR}/test-address.json")
fi
# RecoveryAddress == CW20_TO == tester
RECOVERY_ADDR="${CW20_TO:-${TESTER_ADDR}}"

PRE_SNAP="${ROOT_DIR}/data/balances-pre-fork.json"
if [[ ! -f "${PRE_SNAP}" ]]; then
  echo "ERROR: missing ${PRE_SNAP} (run snapshot-balances.sh pre before halt)"
  exit 1
fi

cw20_bal() {
  local contract="$1"
  local addr="$2"
  local q
  q=$(jq -nc --arg a "${addr}" '{balance:{address:$a}}')
  docker exec -e HOME=/orai localfork-a oraid query wasm contract-state smart "${contract}" "${q}" \
    --home /orai/.oraid --node tcp://127.0.0.1:26657 --output json \
    | jq -r '.data.balance // .balance // "0"'
}

bal_of() {
  curl -sf "${LCD_A}/cosmos/bank/v1beta1/balances/${1}/by_denom?denom=${DENOM}" \
    | jq -r '.balance.amount // "0"'
}

native_bal_of() {
  curl -sf "${LCD_A}/cosmos/bank/v1beta1/balances/${1}/by_denom?denom=${2}" \
    | jq -r '.balance.amount // "0"'
}

HEIGHT_NOW=$(curl -sf "${RPC_A}/status" | jq -r '.result.sync_info.latest_block_height')
ORAI_PRE=$(jq -r --arg a "${RECOVERY_ADDR}" '.wallets[] | select(.address==$a) | .orai // "0"' "${PRE_SNAP}")
CW20_V1_PRE=$(jq -r --arg a "${RECOVERY_ADDR}" '.wallets[] | select(.address==$a) | .cw20_v1 // "0"' "${PRE_SNAP}")
CW20_V2_PRE=$(jq -r --arg a "${RECOVERY_ADDR}" '.wallets[] | select(.address==$a) | .cw20_v2 // "0"' "${PRE_SNAP}")
ORAI_POST=$(bal_of "${RECOVERY_ADDR}")
CW20_V1_POST=$(cw20_bal "${CW20_CONTRACT}" "${RECOVERY_ADDR}")
CW20_V2_POST="0"
if [[ -n "${CW20_CONTRACT_2:-}" ]]; then
  CW20_V2_POST=$(cw20_bal "${CW20_CONTRACT_2}" "${RECOVERY_ADDR}")
fi

echo "  RecoveryAddress=${RECOVERY_ADDR}  height=${HEIGHT_NOW}"
ORAI_WANT=$((ORAI_PRE + POOL_REMAINDER_TO_RECOVERY))
echo "  ORAI     pre=${ORAI_PRE}  post=${ORAI_POST}  want=${ORAI_WANT} (pre + pool_remainder ${POOL_REMAINDER_TO_RECOVERY})"
echo "  CW20 v1  pre=${CW20_V1_PRE}  post=${CW20_V1_POST}  (want ${CW20_AMOUNT})"
echo "  CW20 v2  pre=${CW20_V2_PRE}  post=${CW20_V2_POST}  (want ${CW20_AMOUNT_2:-0})"

REC_FAIL=0
if [[ "${ORAI_POST}" != "${ORAI_WANT}" ]]; then
  echo "ERROR: RecoveryAddress ORAI want ${ORAI_WANT} (pre ${ORAI_PRE} + pool rem ${POOL_REMAINDER_TO_RECOVERY}), got ${ORAI_POST}"
  REC_FAIL=1
fi
if [[ "${CW20_V1_PRE}" != "0" ]]; then
  echo "ERROR: RecoveryAddress CW20 v1 expected 0 before fork, got ${CW20_V1_PRE}"
  REC_FAIL=1
fi
if [[ "${CW20_V1_POST}" != "${CW20_AMOUNT}" ]]; then
  echo "ERROR: RecoveryAddress CW20 v1 want ${CW20_AMOUNT} got ${CW20_V1_POST}"
  REC_FAIL=1
fi
WANT2="${CW20_AMOUNT_2:-0}"
if [[ "${CW20_V2_PRE}" != "0" ]]; then
  echo "ERROR: RecoveryAddress CW20 v2 expected 0 before fork, got ${CW20_V2_PRE}"
  REC_FAIL=1
fi
if [[ "${CW20_V2_POST}" != "${WANT2}" ]]; then
  echo "ERROR: RecoveryAddress CW20 v2 want ${WANT2} got ${CW20_V2_POST}"
  REC_FAIL=1
fi

NATIVE_JSON="${NATIVE_JSON:-${ROOT_DIR}/data/native-denoms.json}"
[[ -f "${NATIVE_JSON}" ]] || NATIVE_JSON="${ROOT_DIR}/native-denoms.json"
NATIVE_ROWS='[]'
if [[ -f "${NATIVE_JSON}" ]]; then
  while IFS=$'\t' read -r NDENOM NAMT; do
    [[ -z "${NDENOM}" ]] && continue
    NPRE=$(jq -r --arg a "${RECOVERY_ADDR}" --arg d "${NDENOM}" \
      '.wallets[] | select(.address==$a) | .native[$d] // "0"' "${PRE_SNAP}")
    NPOST=$(native_bal_of "${RECOVERY_ADDR}" "${NDENOM}")
    echo "  native ${NDENOM}"
    echo "    pre=${NPRE}  post=${NPOST}  (want ${NAMT})"
    if [[ "${NPRE}" != "0" ]]; then
      echo "ERROR: RecoveryAddress native ${NDENOM} expected 0 before fork"
      REC_FAIL=1
    fi
    if [[ "${NPOST}" != "${NAMT}" ]]; then
      echo "ERROR: RecoveryAddress native ${NDENOM} want ${NAMT} got ${NPOST}"
      REC_FAIL=1
    fi
    NATIVE_ROWS=$(jq -nc --argjson rows "${NATIVE_ROWS}" \
      --arg denom "${NDENOM}" --arg pre "${NPRE}" --arg post "${NPOST}" --arg want "${NAMT}" \
      '$rows + [{denom:$denom,pre:$pre,post:$post,want:$want}]')
  done < <(jq -r '.[] | [.denom, .amount] | @tsv' "${NATIVE_JSON}")
fi

jq -nc \
  --arg addr "${RECOVERY_ADDR}" \
  --arg height "${HEIGHT_NOW}" \
  --arg orai_pre "${ORAI_PRE}" --arg orai_post "${ORAI_POST}" --arg orai_want "${ORAI_WANT}" \
  --arg pool_remainder "${POOL_REMAINDER_TO_RECOVERY}" \
  --arg cw20_v1_pre "${CW20_V1_PRE}" --arg cw20_v1_post "${CW20_V1_POST}" --arg cw20_v1_want "${CW20_AMOUNT}" \
  --arg cw20_v2_pre "${CW20_V2_PRE}" --arg cw20_v2_post "${CW20_V2_POST}" --arg cw20_v2_want "${WANT2}" \
  --argjson native "${NATIVE_ROWS}" \
  '{address:$addr,height:($height|tonumber),orai:{pre:$orai_pre,post:$orai_post,want:$orai_want,pool_remainder:$pool_remainder},cw20_v1:{pre:$cw20_v1_pre,post:$cw20_v1_post,want:$cw20_v1_want},cw20_v2:{pre:$cw20_v2_pre,post:$cw20_v2_post,want:$cw20_v2_want},native:$native}' \
  > "${ROOT_DIR}/data/recovery-balances.json"

if [[ "${REC_FAIL}" -ne 0 ]]; then
  exit 1
fi
echo "✓ RecoveryAddress pre→post OK (ORAI += pool remainder; CW20+native rescued)"
echo "  wrote ${ROOT_DIR}/data/recovery-balances.json"

echo "==> Post-fork bank send (tester → node-a → tester)"
NODE_A_ADDR=$(jq -r .address "${ROOT_DIR}/data/node-a/node-a_key.json")
SEND_AMT_NUM=1000000
SEND_AMT="${SEND_AMT_NUM}${DENOM}"
FEE="1000${DENOM}"

tx_ok() {
  local from="$1" to="$2" label="$3"
  echo "  ${label}: ${from} → ${to} (${SEND_AMT})"
  OUT=$(docker exec -e HOME=/orai localfork-a oraid tx bank send "${from}" "${to}" "${SEND_AMT}" \
    --from "${from}" \
    --keyring-backend test \
    --home /orai/.oraid \
    --chain-id "${CHAIN_ID}" \
    --fees "${FEE}" \
    --gas auto --gas-adjustment 1.5 \
    --node tcp://127.0.0.1:26657 \
    --broadcast-mode sync \
    --yes \
    --output json 2>&1) || {
      echo "ERROR: ${label} tx failed"
      echo "${OUT}"
      return 1
    }
  # --gas auto prints "gas estimate: N" on stderr; keep only the JSON object.
  JSON=$(echo "${OUT}" | awk 'BEGIN{f=0} /^\{/{f=1} f{print}' | jq -c . 2>/dev/null | tail -1)
  if [[ -z "${JSON}" ]]; then
    echo "ERROR: ${label} no JSON in tx output"
    echo "${OUT}"
    return 1
  fi
  CODE=$(echo "${JSON}" | jq -r '.code // 0')
  if [[ "${CODE}" != "0" ]]; then
    echo "ERROR: ${label} rejected code=${CODE}"
    echo "${JSON}" | jq .
    return 1
  fi
  TXHASH=$(echo "${JSON}" | jq -r '.txhash // empty')
  echo "  ${label} broadcast ok tx=${TXHASH}"
}

# Expect send to fail (CheckTx/DeliverTx reject, or gas sim error). Confirm recipient bal unchanged.
tx_expect_fail() {
  local from="$1" to="$2" label="$3"
  local bal_to_before bal_to_after
  bal_to_before=$(bal_of "${to}")
  echo "  ${label}: ${from} → ${to} (${SEND_AMT}) [expect FAIL]"
  set +e
  OUT=$(docker exec -e HOME=/orai localfork-a oraid tx bank send "${from}" "${to}" "${SEND_AMT}" \
    --from "${from}" \
    --keyring-backend test \
    --home /orai/.oraid \
    --chain-id "${CHAIN_ID}" \
    --fees "${FEE}" \
    --gas auto --gas-adjustment 1.5 \
    --node tcp://127.0.0.1:26657 \
    --broadcast-mode sync \
    --yes \
    --output json 2>&1)
  RC=$?
  set -e
  JSON=$(echo "${OUT}" | awk 'BEGIN{f=0} /^\{/{f=1} f{print}' | jq -c . 2>/dev/null | tail -1 || true)
  CODE="0"
  if [[ -n "${JSON}" ]]; then
    CODE=$(echo "${JSON}" | jq -r '.code // 0')
  fi
  # Must see the bank restriction error (CLI may also print Usage on failure).
  if ! echo "${OUT}" | grep -Eiq 'is blacklisted|ErrUnauthorized|unauthorized'; then
    echo "ERROR: ${label} did not report blacklist/unauthorized rejection"
    echo "${OUT}"
    return 1
  fi
  if [[ -n "${JSON}" && "${CODE}" == "0" ]] && ! echo "${OUT}" | grep -Eiq 'is blacklisted'; then
    echo "ERROR: ${label} unexpectedly accepted (code=0)"
    echo "${OUT}"
    return 1
  fi
  echo "  ${label} rejected as expected (rc=${RC} code=${CODE})"
  echo "  detail: $(echo "${OUT}" | grep -Ei 'blacklisted|unauthorized' | head -1 | tr '\n' ' ')"

  sleep 3
  bal_to_after=$(bal_of "${to}")
  if [[ "${bal_to_after}" != "${bal_to_before}" ]]; then
    echo "ERROR: ${label} changed recipient balance ${bal_to_before} → ${bal_to_after}"
    return 1
  fi
  echo "  ${label} recipient bal unchanged (${bal_to_after})"
}

BAL_BEFORE=$(bal_of "${TESTER_ADDR}")
echo "  tester bal before=${BAL_BEFORE}"
if [[ "${BAL_BEFORE}" -lt $((SEND_AMT_NUM * 2)) ]]; then
  echo "ERROR: tester underfunded (${BAL_BEFORE})"
  exit 1
fi

sleep 2
tx_ok "${TESTER_KEY}" "${NODE_A_ADDR}" "send-out"
for _ in $(seq 1 20); do
  sleep 1
  BAL_MID=$(bal_of "${TESTER_ADDR}")
  if [[ "${BAL_MID}" -lt "${BAL_BEFORE}" ]]; then
    break
  fi
done
echo "  tester bal after send-out=${BAL_MID}"
if [[ "${BAL_MID}" -ge "${BAL_BEFORE}" ]]; then
  echo "ERROR: send-out did not decrease tester balance"
  exit 1
fi

tx_ok "node-a" "${TESTER_ADDR}" "send-back"
for _ in $(seq 1 20); do
  sleep 1
  BAL_AFTER=$(bal_of "${TESTER_ADDR}")
  if [[ "${BAL_AFTER}" -gt "${BAL_MID}" ]]; then
    break
  fi
done
echo "  tester bal after send-back=${BAL_AFTER}"
if [[ "${BAL_AFTER}" -le "${BAL_MID}" ]]; then
  echo "ERROR: send-back did not increase tester balance"
  exit 1
fi
echo "✓ Post-fork send out + send back succeeded (tester=${TESTER_ADDR})"

echo "==> Post-fork send to blacklist must FAIL"
BL_ADDR="${ADDRS[0]}"
BL_BAL=$(bal_of "${BL_ADDR}")
echo "  blacklist target=${BL_ADDR} bal=${BL_BAL}"
if [[ "${BL_BAL}" != "0" ]]; then
  echo "ERROR: blacklist target should still be burned (0), got ${BL_BAL}"
  exit 1
fi
tx_expect_fail "${TESTER_KEY}" "${BL_ADDR}" "send-to-blacklist"
# Confirm still burned after rejected send
BL_BAL_AFTER=$(bal_of "${BL_ADDR}")
if [[ "${BL_BAL_AFTER}" != "0" ]]; then
  echo "ERROR: blacklist bal became ${BL_BAL_AFTER} after rejected send"
  exit 1
fi
echo "✓ Send to blacklist rejected; balance stays 0"

echo "==> Check RecoveryFrom drained (CW20 + native) after fork"
FROM_BAL=$(cw20_bal "${CW20_CONTRACT}" "${CW20_FROM}")
echo "  CW20 v1 from(${CW20_FROM})=${FROM_BAL} (want 0)"
if [[ "${FROM_BAL}" != "0" ]]; then
  echo "ERROR: RecoveryFrom still holds CW20 v1"
  exit 1
fi
TO2="n/a"
if [[ -n "${CW20_CONTRACT_2:-}" ]]; then
  FROM2=$(cw20_bal "${CW20_CONTRACT_2}" "${CW20_FROM}")
  TO2=$(cw20_bal "${CW20_CONTRACT_2}" "${CW20_TO}")
  echo "  CW20 v2 from=${FROM2} (want 0)"
  if [[ "${FROM2}" != "0" ]]; then
    echo "ERROR: RecoveryFrom still holds CW20 v2"
    exit 1
  fi
fi
TO_BAL=$(cw20_bal "${CW20_CONTRACT}" "${CW20_TO}")
if [[ -f "${NATIVE_JSON}" ]]; then
  while IFS=$'\t' read -r NDENOM _; do
    [[ -z "${NDENOM}" ]] && continue
    FB=$(native_bal_of "${CW20_FROM}" "${NDENOM}")
    echo "  native from ${NDENOM}=${FB} (want 0)"
    if [[ "${FB}" != "0" ]]; then
      echo "ERROR: RecoveryFrom still holds native ${NDENOM}"
      exit 1
    fi
  done < <(jq -r '.[] | [.denom, .amount] | @tsv' "${NATIVE_JSON}")
fi
echo "✓ RecoveryFrom drained (CW20+native=0); RecoveryAddress already asserted above"

echo "==> Check node-b logs for apphash / consensus failure"
sleep 5
if docker logs localfork-b 2>&1 | tail -n 300 | grep -Eiq 'apphash|app hash|Consensus failure|wrong Block.Header.AppHash'; then
  echo "✓ node-b shows apphash/consensus error as expected"
  docker logs localfork-b 2>&1 | tail -n 40 | grep -Ei 'apphash|app hash|Consensus failure|wrong Block.Header.AppHash' || true
else
  echo "WARN: did not find explicit apphash string yet; dumping recent B logs"
  docker logs localfork-b 2>&1 | tail -n 60
  # Soft check: B should NOT show burned balances if it somehow stayed in sync without fork
  B_LCD="${B_LCD:-http://127.0.0.1:1327}"
  B0=$(curl -sf "${B_LCD}/cosmos/bank/v1beta1/balances/${ADDRS[0]}/by_denom?denom=${DENOM}" \
    | jq -r '.balance.amount // empty' || true)
  if [[ -n "${B0}" && "${B0}" == "0" ]]; then
    echo "ERROR: node-b unexpectedly has burned blacklist balance (same fork state as A)"
    exit 1
  fi
  echo "node-b first blacklist bal=${B0:-unavailable} (not burned) — check logs manually if needed"
fi

echo "==> Summary"
echo "  A/S: past ${FORK_HEIGHT}, blacklist ORAI burned"
echo "  revert addresses: trimmed to keep amounts (${REVERT_JSON})"
echo "  RecoveryAddress pre→post: ${ROOT_DIR}/data/recovery-balances.json"
echo "  post-fork bank send out+back: ok (tester=${TESTER_ADDR})"
echo "  post-fork send to blacklist: rejected"
echo "  CW20 rescue: from=0 to_v1=${TO_BAL} to_v2=${TO2:-n/a}"
echo "  native rescue: RecoveryFrom→RecoveryAddress"
echo "  B: old binary / apphash divergence expected"
echo "  blacklist file: ${BLACKLIST_JSON}"
