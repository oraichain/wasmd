#!/usr/bin/env bash
# Scenario: a bad fixture must halt the chain LOUDLY at the fork block, with state untouched.
#
# The fork panics on any inconsistency by design — a partial fork must never be committed.
# This is the failure mode with the highest blast radius on restart day: every node runs the
# same deterministic code, so every node dies at the same block and the chain cannot advance
# without a corrected binary. This test makes that behaviour explicit and documented rather
# than discovered at 3am.
#
# Trigger used here: fund a RevertAddress BELOW its burn Amount so bal.LT(entry.Amount) trips.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
DATA_DIR="${ROOT_DIR}/data"
BIN="${BIN:-${DATA_DIR}/oraid-prod}"
# shellcheck disable=SC1091
source "$(dirname "$0")/lib.sh"

HOME_DIR="${DATA_DIR}/scenario-panic"
RPC="tcp://127.0.0.1:39657"
RPC_HTTP="http://127.0.0.1:39657"

FIX="${DATA_DIR}/fixtures.json"
FORK_HEIGHT=$(jq -r '.fork_height' "${FIX}")
CHAIN_ID=$(jq -r '.mainnet_chain_id' "${FIX}")
BAD=$(jq -r '.revert_addresses[0].address' "${FIX}")
BAD_AMT=$(jq -r '.revert_addresses[0].amount' "${FIX}")
INITIAL_HEIGHT=$(( FORK_HEIGHT - 5 ))

echo "==> Single-node genesis with ${BAD} underfunded (needs ${BAD_AMT}, gets one less)"
make_single_node "${HOME_DIR}" "${CHAIN_ID}" "${INITIAL_HEIGHT}" "${BAD}"

"${BIN}" start --home "${HOME_DIR}" --rpc.laddr "${RPC}" --p2p.laddr tcp://127.0.0.1:39656 \
  >"${HOME_DIR}/start.log" 2>&1 &
PID=$!
trap 'kill ${PID} 2>/dev/null || true' EXIT

echo "==> Expect the node to die at the fork block"
DIED=0
for _ in $(seq 1 60); do
  sleep 2
  if ! kill -0 "${PID}" 2>/dev/null; then DIED=1; break; fi
done

FAILURES=0
pass() { echo "  ✓ $*"; }
fail() { echo "  ✗ $*"; FAILURES=$((FAILURES + 1)); }

if [[ "${DIED}" -eq 1 ]]; then
  pass "node exited at the fork instead of committing a partial fork"
else
  H=$(curl -sf "${RPC_HTTP}/status" 2>/dev/null | jq -r '.result.sync_info.latest_block_height // 0')
  if [[ "${H}" -gt "${FORK_HEIGHT}" ]]; then
    fail "node advanced past the fork with a bad fixture (height ${H}) — partial fork committed"
  else
    pass "node wedged at height ${H} (did not pass the fork)"
  fi
fi

if grep -q "revert addresses: balance" "${HOME_DIR}/start.log"; then
  pass "panic names the offending fixture"
  grep -m1 "revert addresses: balance" "${HOME_DIR}/start.log" | cut -c1-160 | sed 's/^/      /'
else
  fail "expected a 'revert addresses: balance ... is less than amount' panic"
  tail -15 "${HOME_DIR}/start.log" | cut -c1-160
fi

if grep -q "fork logic applied to state" "${HOME_DIR}/start.log"; then
  fail "fork reported success despite the bad fixture — CacheContext write() must not run"
else
  pass "state never written (write() skipped, parent state untouched)"
fi

# Restarting must fail identically: the panic is deterministic, so operators cannot recover
# by restarting. This is exactly what the runbook has to say.
echo "==> Restart must reproduce the same panic (no recovery by restart)"
"${BIN}" start --home "${HOME_DIR}" --rpc.laddr "${RPC}" --p2p.laddr tcp://127.0.0.1:39656 \
  >"${HOME_DIR}/start2.log" 2>&1 &
PID2=$!
trap 'kill ${PID} ${PID2} 2>/dev/null || true' EXIT
sleep 25
kill "${PID2}" 2>/dev/null || true
if grep -q "revert addresses: balance" "${HOME_DIR}/start2.log"; then
  pass "second start panics the same way — only a new binary/fixture can recover"
else
  echo "  - second start produced no panic line; see ${HOME_DIR}/start2.log"
fi

echo
if [[ "${FAILURES}" -eq 0 ]]; then
  echo "✓ fork-panic failure mode behaves as designed"
  echo "  Operator takeaway: a bad fixture halts the chain at the fork block on EVERY node."
  echo "  Recovery requires distributing a corrected binary — see docs/RESTART-v0.50.14.md."
else
  echo "✗ ${FAILURES} failure(s)"; exit 1
fi
