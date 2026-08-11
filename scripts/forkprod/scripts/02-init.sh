#!/usr/bin/env bash
# Build the 3-validator genesis for the production-fixture fork e2e.
#
# Key differences from scripts/localfork:
#   - chain-id is the real MainnetChainID, so isForkChain() lets the fork run
#   - initial_height is set just below the real ForkHeight (118018795), so the unmodified
#     production binary forks a few blocks after start. constants.go is never patched.
#   - blacklist / revert / recovery accounts are funded from the compiled fixtures
#   - the fabricated wasm genesis from 01-bootstrap-contracts.sh is merged in
set -euo pipefail

source "$(dirname "$0")/env.sh"
WASM_GEN="${DATA_DIR}/wasm-genesis.json"

# Blocks between chain start and the fork. Small so the run is quick, but >1 so we can
# observe pre-fork state and prove the height gate.
PRE_FORK_BLOCKS="${PRE_FORK_BLOCKS:-5}"

# Voting power split: A 30%, B 30% (stays on the old binary), S 40%.
STAKE_A="${STAKE_A:-30000000}"
STAKE_B="${STAKE_B:-30000000}"
STAKE_S="${STAKE_S:-40000000}"

for f in "${FIX}" "${WASM_GEN}"; do
  [[ -f "${f}" ]] || { echo "ERROR: ${f} missing — run 01-bootstrap-contracts.sh first"; exit 1; }
done

DENOM=$(jq -r '.denom' "${FIX}")
CHAIN_ID=$(jq -r '.mainnet_chain_id' "${FIX}")
FORK_HEIGHT=$(jq -r '.fork_height' "${FIX}")
INITIAL_HEIGHT=$(( FORK_HEIGHT - PRE_FORK_BLOCKS ))
RECOVERY_FROM=$(jq -r '.recovery_from_address[0]' "${FIX}")
RECOVERY_ADDR=$(jq -r '.recovery_address' "${FIX}")

echo "==> chain_id=${CHAIN_ID} fork_height=${FORK_HEIGHT} initial_height=${INITIAL_HEIGHT}"

rm -rf "${DATA_DIR}/node-a" "${DATA_DIR}/node-b" "${DATA_DIR}/node-s"

init_node() {
  local name=$1 home="${DATA_DIR}/node-$1"
  "${BIN}" init "${name}" --home "${home}" --chain-id "${CHAIN_ID}" >/dev/null 2>&1
  "${BIN}" keys add "${name}" --home "${home}" --keyring-backend test >/dev/null 2>&1
  "${BIN}" keys show "${name}" -a --home "${home}" --keyring-backend test
}

ADDR_A=$(init_node a)
ADDR_B=$(init_node b)
ADDR_S=$(init_node s)

# Tester account: NOT blacklisted, used for post-fork send assertions.
"${BIN}" keys add tester --home "${DATA_DIR}/node-a" --keyring-backend test >/dev/null 2>&1
ADDR_T=$("${BIN}" keys show tester -a --home "${DATA_DIR}/node-a" --keyring-backend test)

cat > "${DATA_DIR}/accounts.env" <<EOF
ADDR_A=${ADDR_A}
ADDR_B=${ADDR_B}
ADDR_S=${ADDR_S}
ADDR_T=${ADDR_T}
CHAIN_ID=${CHAIN_ID}
DENOM=${DENOM}
FORK_HEIGHT=${FORK_HEIGHT}
INITIAL_HEIGHT=${INITIAL_HEIGHT}
RECOVERY_FROM=${RECOVERY_FROM}
RECOVERY_ADDR=${RECOVERY_ADDR}
EOF

GEN="${DATA_DIR}/node-a/config/genesis.json"

add_acct() {
  "${BIN}" genesis add-genesis-account "$1" "$2${DENOM}" \
    --home "${DATA_DIR}/node-a" --keyring-backend test >/dev/null 2>&1
}

echo "==> Fund validators + tester"
add_acct "${ADDR_A}" 100000000
add_acct "${ADDR_B}" 100000000
add_acct "${ADDR_S}" 100000000
add_acct "${ADDR_T}" 500000000

echo "==> Fund blacklist addresses (entire balance is burned at fork)"
while IFS= read -r addr; do
  add_acct "${addr}" 777000000
  echo "  ${addr} <- 777000000"
done < <(jq -r '.blacklist_addresses[]' "${FIX}")

# Revert entries burn exactly Amount and must keep the remainder, so fund Amount + KEEP.
# Funding below Amount would trip the deliberate bal.LT(entry.Amount) panic.
KEEP="${KEEP:-1000000}"
echo "==> Fund revert addresses (Amount + ${KEEP} keep)"
while IFS=$'\t' read -r addr amount; do
  total=$(python3 -c "print(int('${amount}') + ${KEEP})")
  add_acct "${addr}" "${total}"
  echo "  ${addr} <- ${total} (burn ${amount}, keep ${KEEP})"
done < <(jq -r '.revert_addresses[] | [.address, .amount] | @tsv' "${FIX}")

echo "==> Fund RecoveryFromAddress with the native denoms the fork rescues"
NATIVE_FUND="${NATIVE_FUND:-123000000}"
NATIVE_COINS=$(jq -r --arg amt "${NATIVE_FUND}" \
  '[.recovery_native_denoms[] | $amt + .] | join(",")' "${FIX}")
while IFS= read -r addr; do
  "${BIN}" genesis add-genesis-account "${addr}" "${NATIVE_COINS}" \
    --home "${DATA_DIR}/node-a" --keyring-backend test --append >/dev/null 2>&1
  echo "  ${addr} <- ${NATIVE_COINS}"
done < <(jq -r '.recovery_from_address[]' "${FIX}")

echo "==> Gentxs (A=${STAKE_A} B=${STAKE_B} S=${STAKE_S})"
gentx() {
  local name=$1 stake=$2 addr=$3 home="${DATA_DIR}/node-$1" accnum
  # node-a already holds the template genesis; copying it onto itself is an error.
  if [[ "${home}" != "${DATA_DIR}/node-a" ]]; then
    cp "${GEN}" "${home}/config/genesis.json"
  fi
  # gentx signs offline and defaults to account number 0. Only the first genesis account
  # would validate; every other validator must be told its real number or collect-gentxs
  # fails with "signature verification failed ... verify account number".
  accnum=$(jq -r --arg a "${addr}" \
    '.app_state.auth.accounts[] | select(.address == $a) | .account_number' "${GEN}")
  [[ -n "${accnum}" && "${accnum}" != "null" ]] \
    || { echo "ERROR: no genesis account for ${addr}"; exit 1; }
  "${BIN}" genesis gentx "${name}" "${stake}${DENOM}" \
    --home "${home}" --keyring-backend test --chain-id "${CHAIN_ID}" \
    --offline --account-number "${accnum}" --sequence 0 >/dev/null 2>&1
  ls "${home}/config/gentx/"*.json >/dev/null 2>&1 \
    || { echo "ERROR: gentx for node-${name} produced no file"; exit 1; }
  if [[ "${home}" != "${DATA_DIR}/node-a" ]]; then
    cp "${home}/config/gentx/"*.json "${DATA_DIR}/node-a/config/gentx/"
  fi
  echo "  node-${name}: account_number=${accnum} stake=${stake}"
}
gentx a "${STAKE_A}" "${ADDR_A}"
gentx b "${STAKE_B}" "${ADDR_B}"
gentx s "${STAKE_S}" "${ADDR_S}"

"${BIN}" genesis collect-gentxs --home "${DATA_DIR}/node-a" >/dev/null 2>&1

echo "==> Merge fabricated wasm genesis + set initial_height"
# --slurpfile, not --argjson: the wasm bytecode is megabytes and would blow past ARG_MAX.
jq --slurpfile wasm "${WASM_GEN}" --arg ih "${INITIAL_HEIGHT}" \
  '.app_state.wasm = $wasm[0] | .initial_height = $ih' "${GEN}" > "${GEN}.tmp"
mv "${GEN}.tmp" "${GEN}"

jq -e '.app_state.wasm.contracts | length > 0' "${GEN}" >/dev/null \
  || { echo "ERROR: wasm contracts missing from genesis"; exit 1; }

echo "==> Distribute genesis + peer config (local processes, distinct ports)"
ID_A=$("${BIN}" comet show-node-id --home "${DATA_DIR}/node-a" 2>/dev/null | tail -1)
ID_B=$("${BIN}" comet show-node-id --home "${DATA_DIR}/node-b" 2>/dev/null | tail -1)
ID_S=$("${BIN}" comet show-node-id --home "${DATA_DIR}/node-s" 2>/dev/null | tail -1)
PEERS="${ID_A}@127.0.0.1:$(port_p2p a),${ID_B}@127.0.0.1:$(port_p2p b),${ID_S}@127.0.0.1:$(port_p2p s)"

for n in a b s; do
  home="${DATA_DIR}/node-${n}"
  if [[ "${home}" != "${DATA_DIR}/node-a" ]]; then
    cp "${GEN}" "${home}/config/genesis.json"
  fi
  cfg="${home}/config/config.toml"
  app="${home}/config/app.toml"
  sed -i.bak "s|^persistent_peers = .*|persistent_peers = \"${PEERS}\"|" "${cfg}"
  sed -i.bak 's|^addr_book_strict = .*|addr_book_strict = false|' "${cfg}"
  sed -i.bak 's|^allow_duplicate_ip = .*|allow_duplicate_ip = true|' "${cfg}"
  sed -i.bak 's|^timeout_commit = .*|timeout_commit = "1s"|' "${cfg}"
  sed -i.bak "s|^minimum-gas-prices = .*|minimum-gas-prices = \"0${DENOM}\"|" "${app}"
  # Per-node listen addresses so three processes can coexist on one host.
  python3 - "${app}" "$(port_lcd "${n}")" "$(port_grpc "${n}")" "$(port_jsonrpc "${n}")" <<'PY'
import re, sys
path, lcd, grpc, jsonrpc = sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4]
s = open(path).read()
s = re.sub(r'address = "tcp://[^"]*:1317"', f'address = "tcp://127.0.0.1:{lcd}"', s)
s = re.sub(r'address = "localhost:9090"', f'address = "127.0.0.1:{grpc}"', s)
s = re.sub(r'address = "localhost:9091"', f'address = "127.0.0.1:{int(grpc)+1}"', s)
s = re.sub(r'address = "127\.0\.0\.1:8545"', f'address = "127.0.0.1:{jsonrpc}"', s)
s = re.sub(r'ws-address = "127\.0\.0\.1:8546"', f'ws-address = "127.0.0.1:{int(jsonrpc)+1}"', s)
# Enable the LCD (api section) so verify can query balances.
s = re.sub(r'(\[api\][^\[]*?\nenable = )false', r'\1true', s, flags=re.S)
# Leave the EVM JSON-RPC REQUESTED so 06-verify proves the binary's kill switch wins.
s = re.sub(r'(\[json-rpc\][^\[]*?\nenable = )false', r'\1true', s, flags=re.S)
open(path, 'w').write(s)
PY
  rm -f "${cfg}.bak" "${app}.bak"
done


echo "✓ genesis ready"
echo "  validators: A=${ADDR_A} B=${ADDR_B} S=${ADDR_S}"
echo "  tester:     ${ADDR_T}"
echo "  fork at:    ${FORK_HEIGHT} (start ${INITIAL_HEIGHT})"
