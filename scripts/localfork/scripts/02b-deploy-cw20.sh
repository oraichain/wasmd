#!/usr/bin/env bash
# Deploy cw20-base via Instantiate2 (predictable addrs = RecoveryAssets), mint to blacklist.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
REPO_ROOT="$(cd "${ROOT_DIR}/../.." && pwd)"
META_ENV="${ROOT_DIR}/data/meta.env"
if [[ ! -f "${META_ENV}" ]]; then
  echo "ERROR: missing ${META_ENV}"
  echo "  Run ./scripts/01-init.sh (and ./scripts/02-start-old.sh) before CW20 deploy."
  exit 1
fi
# shellcheck disable=SC1091
source "${META_ENV}"

if [[ ! -f "${ROOT_DIR}/data/tester.address" ]]; then
  echo "ERROR: missing ${ROOT_DIR}/data/tester.address (incomplete init)"
  exit 1
fi
if [[ ! -f "${ROOT_DIR}/data/cw20-deployer.address" ]]; then
  echo "ERROR: missing ${ROOT_DIR}/data/cw20-deployer.address (re-run 01-init.sh)"
  exit 1
fi
if ! docker ps --format '{{.Names}}' | grep -qx localfork-a; then
  echo "ERROR: container localfork-a is not running (./scripts/02-start-old.sh)"
  exit 1
fi

RPC_A="${RPC_A:-http://127.0.0.1:26657}"
CW20_WASM="${CW20_WASM:-${REPO_ROOT}/scripts/ibchooks/bytecode/cw20_base.wasm}"
CW20_AMOUNT="${CW20_AMOUNT:-1000000000}"
# txfees / min-gas-price: store wasm needs more than 1000orai at gas~3M
FEE="${CW20_FEE:-100000${DENOM}}"
KEYRING=test

# Must match app/upgrades/v05014 constants (Instantiate2, fixMsg=false).
EXPECTED_DEPLOYER="${EXPECTED_DEPLOYER:-orai19rl4cm2hmr8afy4kldpxz3fka4jguq0a0nm77x}"
EXPECTED_CW20_1="${EXPECTED_CW20_1:-orai18m6m3uy7zknru9tx5jfpadcjc80jp38ngpqwrralm76kj7fgx80q7hky0y}"
EXPECTED_CW20_2="${EXPECTED_CW20_2:-orai10438e2jljstgk2qwuzjzf67sd4al0v7m0lcuskdl4wlhwqmtv73s69lnft}"
SALT1="${SALT1:-localfork-cw20-v1}"
SALT2="${SALT2:-localfork-cw20-v2}"

if [[ ! -f "${CW20_WASM}" ]]; then
  echo "ERROR: CW20 wasm not found: ${CW20_WASM}"
  exit 1
fi

CW20_FROM="${CW20_FROM:-orai1hru4a5w0c29wr36l2dgaymqqd4h0vju9tlvk8w}"
# Always use this run's tester — do not inherit stale CW20_TO from the parent env.
TESTER_ADDR="$(cat "${ROOT_DIR}/data/tester.address")"
CW20_TO="${TESTER_ADDR}"
CW20_DEPLOYER_KEY=cw20-deployer
CW20_DEPLOYER_ADDR="$(cat "${ROOT_DIR}/data/cw20-deployer.address")"

if [[ "${CW20_DEPLOYER_ADDR}" != "${EXPECTED_DEPLOYER}" ]]; then
  echo "ERROR: cw20-deployer ${CW20_DEPLOYER_ADDR} != expected ${EXPECTED_DEPLOYER}"
  exit 1
fi

echo "==> Wait for chain to produce blocks before CW20 deploy"
H=0
for _ in $(seq 1 60); do
  H=$(curl -sf "${RPC_A}/status" 2>/dev/null | jq -r '.result.sync_info.latest_block_height // 0' 2>/dev/null || echo 0)
  if [[ "${H}" =~ ^[0-9]+$ ]] && [[ "${H}" -ge 2 ]]; then
    echo "  height=${H}"
    break
  fi
  sleep 1
done
if [[ ! "${H}" =~ ^[0-9]+$ ]] || [[ "${H}" -lt 2 ]]; then
  echo "ERROR: chain did not reach height >= 2 (last=${H})"
  exit 1
fi

echo "==> Copy wasm into localfork-a"
docker cp "${CW20_WASM}" localfork-a:/tmp/cw20.wasm

tx() {
  docker exec -e HOME=/orai localfork-a oraid "$@" \
    --keyring-backend "${KEYRING}" \
    --home /orai/.oraid \
    --chain-id "${CHAIN_ID}" \
    --fees "${FEE}" \
    --gas auto --gas-adjustment 1.5 \
    --node tcp://127.0.0.1:26657 \
    --broadcast-mode sync \
    --yes \
    --output json 2>&1
}

wait_tx() {
  local hash="$1"
  local i out code
  for i in $(seq 1 60); do
    out=$(docker exec -e HOME=/orai localfork-a oraid query tx "${hash}" \
      --home /orai/.oraid --node tcp://127.0.0.1:26657 --output json 2>/dev/null || true)
    if [[ -n "${out}" ]] && echo "${out}" | jq -e '.height != "0" and .height != null' >/dev/null 2>&1; then
      code=$(echo "${out}" | jq -r '.code // 0')
      if [[ "${code}" != "0" ]]; then
        echo "ERROR: tx ${hash} failed code=${code}" >&2
        echo "${out}" | jq . >&2 || true
        return 1
      fi
      echo "${out}"
      return 0
    fi
    sleep 1
  done
  echo "ERROR: timed out waiting for tx ${hash}" >&2
  return 1
}

parse_json() {
  echo "$1" | awk 'BEGIN{f=0} /^\{/{f=1} f{print}' | jq -c . 2>/dev/null | tail -1
}

require_broadcast_ok() {
  local label="$1"
  local json="$2"
  local code hash
  code=$(echo "${json}" | jq -r '.code // 0')
  hash=$(echo "${json}" | jq -r '.txhash // empty')
  if [[ -z "${hash}" || "${hash}" == "null" ]]; then
    echo "ERROR: ${label}: missing txhash" >&2
    echo "${json}" >&2
    exit 1
  fi
  if [[ "${code}" != "0" ]]; then
    echo "ERROR: ${label}: broadcast code=${code} tx=${hash}" >&2
    echo "${json}" | jq -r '.raw_log // .' >&2
    exit 1
  fi
  echo "${hash}"
}

contract_from_tx() {
  echo "$1" | jq -r '
    [.events[] | select(.type=="instantiate") | .attributes[]
      | select(.key=="_contract_address" or .key=="contract_address") | .value] | first
  '
}

echo "==> Store CW20 code (from ${CW20_DEPLOYER_KEY})"
STORE_OUT=$(tx tx wasm store /tmp/cw20.wasm --from "${CW20_DEPLOYER_KEY}")
STORE_JSON=$(parse_json "${STORE_OUT}")
STORE_HASH=$(require_broadcast_ok "store" "${STORE_JSON}")
echo "  store tx=${STORE_HASH}"
STORE_RES=$(wait_tx "${STORE_HASH}")
CODE_ID=$(echo "${STORE_RES}" | jq -r '.events[] | select(.type=="store_code") | .attributes[] | select(.key=="code_id") | .value' | head -1)
if [[ -z "${CODE_ID}" || "${CODE_ID}" == "null" ]]; then
  CODE_ID=$(echo "${STORE_RES}" | jq -r '
    [.events[] | select(.type=="store_code") | .attributes[] | select(.key=="code_id" or .key=="Y29kZV9pZA==") | .value] | first
  ')
fi
echo "  code_id=${CODE_ID}"
if [[ -z "${CODE_ID}" || "${CODE_ID}" == "null" ]]; then
  echo "ERROR: could not parse code_id"
  echo "${STORE_RES}" | jq .
  exit 1
fi

INIT_MSG=$(jq -nc \
  --arg minter "${CW20_DEPLOYER_ADDR}" \
  '{name:"LocalforkCW20",symbol:"LFC",decimals:6,initial_balances:[],mint:{minter:$minter}}')

instantiate2() {
  # Prints only the contract address on stdout; logs go to stderr.
  local salt="$1"
  local label="$2"
  local expected="$3"
  echo "==> Instantiate2 salt=${salt} (expect ${expected})" >&2
  local out json hash res addr
  out=$(tx tx wasm instantiate2 "${CODE_ID}" "${INIT_MSG}" "${salt}" \
    --ascii \
    --from "${CW20_DEPLOYER_KEY}" \
    --label "${label}" \
    --admin "${CW20_DEPLOYER_ADDR}")
  json=$(parse_json "${out}")
  hash=$(require_broadcast_ok "instantiate2 ${salt}" "${json}")
  echo "  instantiate2 tx=${hash}" >&2
  res=$(wait_tx "${hash}")
  addr=$(contract_from_tx "${res}")
  if [[ -z "${addr}" || "${addr}" == "null" ]]; then
    addr=$(docker exec -e HOME=/orai localfork-a oraid query wasm list-contract-by-code "${CODE_ID}" \
      --home /orai/.oraid --node tcp://127.0.0.1:26657 --output json | jq -r '.contracts | last')
  fi
  echo "  contract=${addr}" >&2
  if [[ "${addr}" != "${expected}" ]]; then
    echo "ERROR: Instantiate2 addr ${addr} != expected RecoveryAssets ${expected}" >&2
    echo "  (checksum/creator/salt mismatch vs app/upgrades/v05014.RecoveryAssets)" >&2
    exit 1
  fi
  printf '%s\n' "${addr}"
}

CW20_CONTRACT=$(instantiate2 "${SALT1}" "localfork-cw20-v1" "${EXPECTED_CW20_1}")
CW20_CONTRACT_2=$(instantiate2 "${SALT2}" "localfork-cw20-v2" "${EXPECTED_CW20_2}")

MINT_MSG=$(jq -nc --arg r "${CW20_FROM}" --arg a "${CW20_AMOUNT}" \
  '{mint:{recipient:$r,amount:$a}}')
echo "==> Mint ${CW20_AMOUNT} on ${CW20_CONTRACT} → blacklist ${CW20_FROM}"
MINT_OUT=$(tx tx wasm execute "${CW20_CONTRACT}" "${MINT_MSG}" --from "${CW20_DEPLOYER_KEY}")
MINT_JSON=$(parse_json "${MINT_OUT}")
MINT_HASH=$(require_broadcast_ok "mint" "${MINT_JSON}")
echo "  mint tx=${MINT_HASH}"
wait_tx "${MINT_HASH}" >/dev/null

BAL_Q=$(jq -nc --arg a "${CW20_FROM}" '{balance:{address:$a}}')
BAL=$(docker exec -e HOME=/orai localfork-a oraid query wasm contract-state smart "${CW20_CONTRACT}" "${BAL_Q}" \
  --home /orai/.oraid --node tcp://127.0.0.1:26657 --output json | jq -r '.data.balance // .balance // empty')
echo "  blacklist cw20 bal=${BAL}"
if [[ "${BAL}" != "${CW20_AMOUNT}" ]]; then
  echo "ERROR: expected mint balance ${CW20_AMOUNT}, got ${BAL}"
  exit 1
fi

cat > "${ROOT_DIR}/data/cw20.env" <<EOF
CW20_CONTRACT=${CW20_CONTRACT}
CW20_CONTRACT_2=${CW20_CONTRACT_2}
CW20_FROM=${CW20_FROM}
CW20_TO=${CW20_TO}
CW20_AMOUNT=${CW20_AMOUNT}
CW20_CODE_ID=${CODE_ID}
CW20_DEPLOYER=${CW20_DEPLOYER_ADDR}
EOF

echo "✓ CW20 Instantiate2 deployed; wrote ${ROOT_DIR}/data/cw20.env"
cat "${ROOT_DIR}/data/cw20.env"
