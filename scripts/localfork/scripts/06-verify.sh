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

echo "==> Post-fork bank send (tester → node-a → tester)"
TESTER_KEY="${TESTER_KEY:-tester}"
TESTER_ADDR="${TESTER_ADDR:-$(cat "${ROOT_DIR}/data/tester.address" 2>/dev/null || true)}"
if [[ -z "${TESTER_ADDR}" ]]; then
  TESTER_ADDR=$(jq -r .address "${ROOT_DIR}/test-address.json")
fi
NODE_A_ADDR=$(jq -r .address "${ROOT_DIR}/data/node-a/node-a_key.json")
SEND_AMT_NUM=1000000
SEND_AMT="${SEND_AMT_NUM}${DENOM}"
FEE="1000${DENOM}"

bal_of() {
  curl -sf "${LCD_A}/cosmos/bank/v1beta1/balances/${1}/by_denom?denom=${DENOM}" \
    | jq -r '.balance.amount // "0"'
}

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

echo "==> Check CW20 rescue state after fork (RecoveryAssets via Instantiate2)"
CW20_ENV="${ROOT_DIR}/data/cw20.env"
if [[ ! -f "${CW20_ENV}" ]]; then
  echo "ERROR: missing ${CW20_ENV} (run 02b-deploy-cw20.sh)"
  exit 1
fi
# shellcheck disable=SC1090
source "${CW20_ENV}"

cw20_bal() {
  local contract="$1"
  local addr="$2"
  local q
  q=$(jq -nc --arg a "${addr}" '{balance:{address:$a}}')
  docker exec -e HOME=/orai localfork-a oraid query wasm contract-state smart "${contract}" "${q}" \
    --home /orai/.oraid --node tcp://127.0.0.1:26657 --output json \
    | jq -r '.data.balance // .balance // "0"'
}

FROM_BAL=$(cw20_bal "${CW20_CONTRACT}" "${CW20_FROM}")
TO_BAL=$(cw20_bal "${CW20_CONTRACT}" "${CW20_TO}")
echo "  RecoveryAssets[0]=${CW20_CONTRACT}"
echo "  from(${CW20_FROM}) bal=${FROM_BAL}"
echo "  to(${CW20_TO}) bal=${TO_BAL} (want ${CW20_AMOUNT})"
if [[ "${FROM_BAL}" != "0" ]]; then
  echo "ERROR: blacklist still holds CW20 after fork rescue"
  exit 1
fi
if [[ "${TO_BAL}" != "${CW20_AMOUNT}" ]]; then
  echo "ERROR: tester CW20 bal want ${CW20_AMOUNT} got ${TO_BAL}"
  exit 1
fi
# Second Instantiate2 contract was deployed empty — rescue should no-op (still 0/0).
if [[ -n "${CW20_CONTRACT_2:-}" ]]; then
  FROM2=$(cw20_bal "${CW20_CONTRACT_2}" "${CW20_FROM}")
  TO2=$(cw20_bal "${CW20_CONTRACT_2}" "${CW20_TO}")
  echo "  RecoveryAssets[1]=${CW20_CONTRACT_2} from=${FROM2} to=${TO2}"
  if [[ "${FROM2}" != "0" || "${TO2}" != "0" ]]; then
    echo "ERROR: unexpected balances on empty second RecoveryAsset"
    exit 1
  fi
fi
echo "✓ CW20 rescued: blacklist=0, tester=${TO_BAL}"

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
echo "  post-fork bank send out+back: ok (tester=${TESTER_ADDR})"
echo "  post-fork send to blacklist: rejected"
echo "  CW20 rescue: from=0 to=${TO_BAL} RecoveryAssets[0]=${CW20_CONTRACT}"
echo "  B: old binary / apphash divergence expected"
echo "  blacklist file: ${BLACKLIST_JSON}"
