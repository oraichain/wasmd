#!/usr/bin/env bash
# Scenario: production binary on a NON-mainnet chain-id must SKIP the fork.
#
# The production fixtures carry real mainnet addresses and amounts. isForkChain() refuses to
# apply them anywhere except chain-id Oraichain — and skips rather than panics, so a devnet
# running this binary keeps producing blocks.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
DATA_DIR="${ROOT_DIR}/data"
BIN="${BIN:-${DATA_DIR}/oraid-prod}"
# shellcheck disable=SC1091
source "$(dirname "$0")/lib.sh"

HOME_DIR="${DATA_DIR}/scenario-chainid"
RPC="tcp://127.0.0.1:38657"
RPC_HTTP="http://127.0.0.1:38657"
WRONG_CHAIN_ID="not-oraichain"

FIX="${DATA_DIR}/fixtures.json"
FORK_HEIGHT=$(jq -r '.fork_height' "${FIX}")
DENOM=$(jq -r '.denom' "${FIX}")
BLK=$(jq -r '.blacklist_addresses[0]' "${FIX}")
INITIAL_HEIGHT=$(( FORK_HEIGHT - 5 ))

echo "==> Build single-node genesis on chain-id=${WRONG_CHAIN_ID} (fork at ${FORK_HEIGHT})"
make_single_node "${HOME_DIR}" "${WRONG_CHAIN_ID}" "${INITIAL_HEIGHT}"

"${BIN}" start --home "${HOME_DIR}" --rpc.laddr "${RPC}" --p2p.laddr tcp://127.0.0.1:38656 \
  >"${HOME_DIR}/start.log" 2>&1 &
PID=$!
trap 'kill ${PID} 2>/dev/null || true' EXIT

echo "==> Wait for the chain to pass ForkHeight"
H=0
for _ in $(seq 1 90); do
  sleep 2
  H=$(curl -sf "${RPC_HTTP}/status" 2>/dev/null | jq -r '.result.sync_info.latest_block_height // 0')
  [[ "${H}" -gt "${FORK_HEIGHT}" ]] && break
done

FAILURES=0
pass() { echo "  ✓ $*"; }
fail() { echo "  ✗ $*"; FAILURES=$((FAILURES + 1)); }

if [[ "${H}" -gt "${FORK_HEIGHT}" ]]; then
  pass "chain kept producing past the fork (height ${H}) — skip, not halt"
else
  fail "chain stalled at ${H}; expected it to sail past ${FORK_HEIGHT}"
  tail -20 "${HOME_DIR}/start.log"
fi

if grep -q "fork logic skipped" "${HOME_DIR}/start.log"; then
  pass "guard logged the skip"
  grep -m1 "fork logic skipped" "${HOME_DIR}/start.log" | sed 's/^/      /'
else
  fail "no 'fork logic skipped' message in the log"
fi

if grep -q "running v0.50.14 fork logic" "${HOME_DIR}/start.log" \
   && grep -q "fork logic applied to state" "${HOME_DIR}/start.log"; then
  fail "fork logic was APPLIED on ${WRONG_CHAIN_ID} — mainnet burns leaked onto a test chain"
else
  pass "fork logic never applied state"
fi

B=$(curl -sf "http://127.0.0.1:1317/cosmos/bank/v1beta1/balances/${BLK}/by_denom?denom=${DENOM}" 2>/dev/null | jq -r '.balance.amount // "unknown"')
if [[ "${B}" == "777000000" ]]; then
  pass "blacklist balance untouched (${B})"
elif [[ "${B}" == "unknown" ]]; then
  echo "  - LCD disabled; skipping balance check (log assertions above are authoritative)"
else
  fail "blacklist balance is ${B}, expected the genesis 777000000 (fork must not have run)"
fi

echo
[[ "${FAILURES}" -eq 0 ]] && echo "✓ chain-id guard holds" || { echo "✗ ${FAILURES} failure(s)"; exit 1; }
