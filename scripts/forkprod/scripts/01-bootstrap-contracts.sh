#!/usr/bin/env bash
# Fabricate the wasm genesis fragment the production fork fixtures require.
#
# The production constants reference real mainnet contract addresses. Those are
# Instantiate2-derived from the real creator/salt/checksum and cannot be reproduced on a
# fresh chain, so instead we: instantiate the same contract code on a throwaway node,
# `export` the genesis, and rewrite contract_address to the production addresses.
#
# Safe for these contracts because none of them store their own address in state (asserted
# below). Produces data/wasm-genesis.json for 02-init.sh.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
REPO_ROOT="$(cd "${ROOT_DIR}/../.." && pwd)"
DATA_DIR="${ROOT_DIR}/data"
TESTDATA="${REPO_ROOT}/app/upgrades/v05014/testdata"
FIX="${DATA_DIR}/fixtures.json"

BOOT_HOME="${DATA_DIR}/bootstrap"
RPC="tcp://127.0.0.1:37657"
RPC_HTTP="http://127.0.0.1:37657"
CHAIN_ID="Oraichain"

mkdir -p "${DATA_DIR}"

# The oraid binary prints a `WARNING:(ast) sonic ...` banner on some toolchains, which
# corrupts any JSON capture. Strip everything before the first JSON line.
json() { sed -n '/^[[{]/,$p'; }

echo "==> Dump production fixtures from the compiled constants"
(cd "${REPO_ROOT}" && go run ./scripts/forkprod/fixtures 2>/dev/null) > "${FIX}"
jq -e '.is_localfork_build == false' "${FIX}" >/dev/null \
  || { echo "ERROR: fixtures came from a localfork build"; exit 1; }

DENOM=$(jq -r '.denom' "${FIX}")
ADMIN=$(jq -r '.admin_contract' "${FIX}")
POOL_V2=$(jq -r '.pause_pool_v2' "${FIX}")
POOL_V3=$(jq -r '.pause_pool_v3' "${FIX}")
RECOVERY_FROM=$(jq -r '.recovery_from_address[0]' "${FIX}")
CW20_TARGETS=(); while IFS= read -r a; do CW20_TARGETS+=("${a}"); done < <(jq -r '.recovery_assets[]' "${FIX}")

# Binary: prefer the image build, fall back to a local go build for speed.
BIN="${BIN:-${DATA_DIR}/oraid-prod}"
if [[ ! -x "${BIN}" ]]; then
  echo "==> Building local production binary for bootstrap"
  (cd "${REPO_ROOT}" && go build -o "${BIN}" ./cmd/wasmd)
fi

echo "==> Init throwaway bootstrap chain"
rm -rf "${BOOT_HOME}"
"${BIN}" init bootstrap --home "${BOOT_HOME}" --chain-id "${CHAIN_ID}" >/dev/null 2>&1
"${BIN}" keys add dep --home "${BOOT_HOME}" --keyring-backend test >/dev/null 2>&1
DEP=$("${BIN}" keys show dep -a --home "${BOOT_HOME}" --keyring-backend test)
"${BIN}" genesis add-genesis-account "${DEP}" "1000000000000000${DENOM}" \
  --home "${BOOT_HOME}" --keyring-backend test >/dev/null 2>&1
"${BIN}" genesis gentx dep "10000000000${DENOM}" \
  --home "${BOOT_HOME}" --keyring-backend test --chain-id "${CHAIN_ID}" >/dev/null 2>&1
"${BIN}" genesis collect-gentxs --home "${BOOT_HOME}" >/dev/null 2>&1
sed -i.bak 's|^minimum-gas-prices = .*|minimum-gas-prices = "0orai"|' "${BOOT_HOME}/config/app.toml"

"${BIN}" start --home "${BOOT_HOME}" \
  --rpc.laddr "${RPC}" --p2p.laddr tcp://127.0.0.1:37656 \
  >"${BOOT_HOME}/start.log" 2>&1 &
BOOT_PID=$!
trap 'kill ${BOOT_PID} 2>/dev/null || true' EXIT

for _ in $(seq 1 60); do
  sleep 1
  curl -sf "${RPC_HTTP}/status" >/dev/null 2>&1 && break
done
sleep 4

# Broadcast and assert the tx actually committed. A silent failure here would produce a
# genesis missing contracts and make every downstream assertion meaningless.
tx() {
  local out hash code rc
  set +e
  out=$("${BIN}" tx "$@" --from dep --home "${BOOT_HOME}" --keyring-backend test \
    --chain-id "${CHAIN_ID}" --node "${RPC}" --gas 8000000 --gas-prices "0.001${DENOM}" \
    -y -o json 2>"${BOOT_HOME}/tx.err")
  rc=$?
  set -e
  hash=$(echo "${out}" | json | jq -r '.txhash // empty' 2>/dev/null)
  if [[ -z "${hash}" ]]; then
    echo "ERROR: broadcast failed (rc=${rc})"
    echo "  stdout: ${out}"
    echo "  stderr: $(cat "${BOOT_HOME}/tx.err")"
    exit 1
  fi
  code=""
  for _ in $(seq 1 30); do
    sleep 1
    code=$(q tx "${hash}" 2>/dev/null | jq -r '.code // empty' || true)
    if [[ -n "${code}" ]]; then break; fi
  done
  if [[ "${code}" != "0" ]]; then
    echo "ERROR: tx ${hash} failed (code=${code:-not-included})"
    q tx "${hash}" 2>/dev/null | jq -r '.raw_log' | head -3 || true
    exit 1
  fi
}
q() { "${BIN}" query "$@" --home "${BOOT_HOME}" --node "${RPC}" -o json 2>/dev/null | json; }

echo "==> Store contract code"
tx wasm store "${TESTDATA}/cw20_base.wasm"
tx wasm store "${TESTDATA}/oraiswap-pair.wasm"
tx wasm store "${TESTDATA}/oraiswap-v3.wasm"
CW20_CODE=1; PAIR_CODE=2; V3_CODE=3

echo "==> Instantiate one cw20 per RecoveryAssets entry (balance owned by RecoveryFromAddress)"
# cw20_base enforces ticker [a-zA-Z\-]{3,12} — no digits.
LETTERS=ABCDEFGHIJKLMNOPQRSTUVWXYZ
# Capture each address right after its own instantiate. The oraiswap-pair below also
# instantiates an LP token from the same code id, so selecting by code id at the end would
# depend on ordering luck.
CW20_SRC=()
for i in "${!CW20_TARGETS[@]}"; do
  INIT=$(jq -nc --arg to "${RECOVERY_FROM}" --arg sym "MOCK${LETTERS:$i:1}" \
    '{name:"MockRecovery",symbol:$sym,decimals:6,initial_balances:[{address:$to,amount:"1000000000"}]}')
  tx wasm instantiate "${CW20_CODE}" "${INIT}" --label "cw20-${i}" --no-admin
  CW20_SRC+=("$(q wasm list-contract-by-code "${CW20_CODE}" | jq -r '.contracts[-1]')")
done

echo "==> Instantiate oraiswap-pair (admin = production AdminContract)"
PAIR_INIT=$(jq -nc --argjson code "${CW20_CODE}" --arg admin "${ADMIN}" --arg denom "${DENOM}" \
  '{token_code_id:$code,oracle_addr:$admin,
    asset_infos:[{native_token:{denom:$denom}},{native_token:{denom:"ibc/MOCKPAIRDENOM"}}],
    admin:$admin}')
tx wasm instantiate "${PAIR_CODE}" "${PAIR_INIT}" --label pair --no-admin

echo "==> Instantiate oraiswap-v3 (admin = production AdminContract)"
V3_INIT=$(jq -nc --arg admin "${ADMIN}" '{protocol_fee:250000000000,incentives_fund_manager:$admin}')
tx wasm instantiate "${V3_CODE}" "${V3_INIT}" --label v3 --no-admin

PAIR_SRC=$(q wasm list-contract-by-code "${PAIR_CODE}" | jq -r '.contracts[0]')
V3_SRC=$(q wasm list-contract-by-code "${V3_CODE}" | jq -r '.contracts[0]')

if [[ "${#CW20_SRC[@]}" -lt "${#CW20_TARGETS[@]}" ]]; then
  echo "ERROR: instantiated ${#CW20_SRC[@]} cw20 contracts, need ${#CW20_TARGETS[@]}"
  exit 1
fi
echo "  cw20 sources: ${CW20_SRC[*]}"
echo "  pair: ${PAIR_SRC}"
echo "  v3:   ${V3_SRC}"

echo "==> Export and rewrite contract addresses to production values"
kill "${BOOT_PID}" 2>/dev/null || true
wait "${BOOT_PID}" 2>/dev/null || true
trap - EXIT
sleep 2
"${BIN}" export --home "${BOOT_HOME}" > "${DATA_DIR}/bootstrap-export.json"

MAP="${DATA_DIR}/addr-map.json"
{
  echo '{'
  for i in "${!CW20_TARGETS[@]}"; do
    printf '"%s":"%s",' "${CW20_SRC[$i]}" "${CW20_TARGETS[$i]}"
  done
  printf '"%s":"%s",' "${PAIR_SRC}" "${POOL_V2}"
  printf '"%s":"%s"' "${V3_SRC}" "${POOL_V3}"
  echo '}'
} > "${MAP}"

# Guard: none of these contracts may embed their own address in state, or rewriting
# silently corrupts them.
for src in "${CW20_SRC[@]}" "${PAIR_SRC}" "${V3_SRC}"; do
  HITS=$(jq -r --arg a "${src}" '
    .app_state.wasm.contracts[] | select(.contract_address == $a)
    | .contract_state[].value' "${DATA_DIR}/bootstrap-export.json" \
    | base64 -d 2>/dev/null | grep -c "${src}" || true)
  if [[ "${HITS}" != "0" ]]; then
    echo "ERROR: contract ${src} embeds its own address in state (${HITS} hits) — rewrite unsafe"
    exit 1
  fi
done
echo "  ✓ no contract embeds its own address"

jq --argjson m "$(cat "${MAP}")" '
  .app_state.wasm
  | .contracts = [ .contracts[]
      | .contract_address = ( ($m[.contract_address]) // .contract_address ) ]
' "${DATA_DIR}/bootstrap-export.json" > "${DATA_DIR}/wasm-genesis.json"

# The pool contracts record their admin from info.sender at instantiate (oraiswap-v3) or
# from an init field (oraiswap-pair). On mainnet that admin IS AdminContract, and the fork
# executes pause/enable_whitelist as AdminContract — so the fixture has to say the same, or
# the pool answers Unauthorized. We cannot instantiate from a contract address, so rewrite
# the recorded admin in the exported state.
python3 - "${DATA_DIR}/wasm-genesis.json" "${DEP}" "${ADMIN}" "${POOL_V2}" "${POOL_V3}" <<'PY'
import base64, json, sys
path, dep, admin, pool_v2, pool_v3 = sys.argv[1:6]
g = json.load(open(path))
pools = {pool_v2, pool_v3}
changed = 0
for c in g["contracts"]:
    if c["contract_address"] not in pools:
        continue
    c["contract_info"]["creator"] = admin
    for e in c.get("contract_state", []):
        raw = base64.b64decode(e["value"])
        try:
            txt = raw.decode("utf8")
        except UnicodeDecodeError:
            continue
        if dep in txt:
            e["value"] = base64.b64encode(txt.replace(dep, admin).encode()).decode()
            changed += 1
json.dump(g, open(path, "w"))
print(f"  ✓ rewrote admin -> AdminContract in {changed} pool state entries")
PY

echo "==> Rewritten contract addresses:"
jq -r '.contracts[].contract_address' "${DATA_DIR}/wasm-genesis.json" | sed 's/^/  /'

for want in "${CW20_TARGETS[@]}" "${POOL_V2}" "${POOL_V3}"; do
  jq -e --arg a "${want}" 'any(.contracts[]; .contract_address == $a)' \
    "${DATA_DIR}/wasm-genesis.json" >/dev/null \
    || { echo "ERROR: ${want} missing from rewritten wasm genesis"; exit 1; }
done

echo "✓ wasm genesis fragment ready: ${DATA_DIR}/wasm-genesis.json"
