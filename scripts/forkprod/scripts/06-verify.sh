#!/usr/bin/env bash
# Verify the production-fixture fork on a live 3-validator chain.
#
# Checks, in order:
#   1. A + S progressed past the real ForkHeight
#   2. every BlacklistAddresses balance is zero (burned)
#   3. every RevertAddress burned exactly Amount and kept the remainder
#   4. RecoveryFromAddress drained of CW20 + native denoms, RecoveryAddress credited
#   5. pools paused (oraiswap-pair whitelist enabled, oraiswap-v3 paused)
#   6. fork wired the blacklist into the txfees store; inbound still allowed (freezes)
#      NOTE: outbound blocking is NOT e2e-testable here -- see the scope note at step 6
#   7. normal (non-blacklisted) transfers still work
#   8. EVM is dead: no JSON-RPC listener despite app.toml enable=true
#   9. node-b (stale binary) diverged instead of following
set -euo pipefail

source "$(dirname "$0")/env.sh"
# shellcheck disable=SC1091
source "${DATA_DIR}/accounts.env"

RPC_A=$(rpc_url a); RPC_S=$(rpc_url s); LCD_A=$(lcd_url a)
JSONRPC_A="http://127.0.0.1:$(port_jsonrpc a)"
HOME_A=$(node_home a)
NODE_A="tcp://127.0.0.1:$(port_rpc a)"

FAILURES=0
pass() { echo "  ✓ $*"; }
fail() { echo "  ✗ $*"; FAILURES=$((FAILURES + 1)); }

bal() { # addr [denom]  -- current balance
  curl -sf "${LCD_A}/cosmos/bank/v1beta1/balances/$1/by_denom?denom=${2:-${DENOM}}" \
    | jq -r '.balance.amount // "0"'
}
# Balance AT the fork height. The burn assertions mean "the fork zeroed this", so pin them
# to that height -- otherwise step 6 (which funds a blacklisted account to prove inbound is
# still allowed) would make a re-run of this script fail steps 2/3.
bal_at() { # addr height [denom]
  curl -sf -H "x-cosmos-block-height: $2" \
    "${LCD_A}/cosmos/bank/v1beta1/balances/$1/by_denom?denom=${3:-${DENOM}}" \
    | jq -r '.balance.amount // "0"'
}
smart() { # contract json
  "${BIN_NEW}" query wasm contract-state smart "$1" "$2" \
    --home "${HOME_A}" --node "${NODE_A}" -o json 2>/dev/null | json
}
otx() { "${BIN_NEW}" tx "$@" --home "${HOME_A}" --keyring-backend test \
  --chain-id "${CHAIN_ID}" --node "${NODE_A}" --gas 400000 --gas-prices "0.001${DENOM}" -y -o json; }
oq()  { "${BIN_NEW}" query "$@" --home "${HOME_A}" --node "${NODE_A}" -o json 2>/dev/null | json; }

echo "==> 1. Wait for A to pass ForkHeight ${FORK_HEIGHT}"
for _ in $(seq 1 90); do
  HA=$(curl -sf "${RPC_A}/status" 2>/dev/null | jq -r '.result.sync_info.latest_block_height // 0')
  [[ "${HA}" -gt "${FORK_HEIGHT}" ]] && break
  sleep 2
done
HA=$(curl -sf "${RPC_A}/status" | jq -r '.result.sync_info.latest_block_height')
HS=$(curl -sf "${RPC_S}/status" | jq -r '.result.sync_info.latest_block_height')
if [[ "${HA}" -gt "${FORK_HEIGHT}" && "${HS}" -gt "${FORK_HEIGHT}" ]]; then
  pass "A=${HA} S=${HS} both past fork ${FORK_HEIGHT}"
else
  fail "A=${HA} S=${HS} did not pass fork ${FORK_HEIGHT}"
  echo "--- node-a log tail ---"; tail -25 "${HOME_A}/start.log"
  exit 1
fi

echo "==> 2. Blacklist balances burned"
while IFS= read -r addr; do
  B=$(bal_at "${addr}" "${FORK_HEIGHT}")
  [[ "${B}" == "0" ]] && pass "${addr} = 0 at fork height" \
    || fail "${addr} = ${B} at fork height, expected 0"
done < <(jq -r '.blacklist_addresses[]' "${FIX}")

echo "==> 3. Revert addresses burned exactly Amount"
KEEP="${KEEP:-1000000}"
while IFS=$'\t' read -r addr amount; do
  B=$(bal_at "${addr}" "${FORK_HEIGHT}")
  # Pool address is special-cased in fork.go: remainder is swept to RecoveryAddress.
  if [[ "${addr}" == "orai15aunrryk5yqsrgy0tvzpj7pupu62s0t2n09t0dscjgzaa27e44esefzgf8" ]]; then
    [[ "${B}" == "0" ]] && pass "pool ${addr} swept to recovery (0)" \
      || fail "pool ${addr} = ${B}, expected 0"
  else
    [[ "${B}" == "${KEEP}" ]] && pass "${addr} kept ${KEEP}" \
      || fail "${addr} = ${B}, expected ${KEEP} (burn ${amount})"
  fi
done < <(jq -r '.revert_addresses[] | [.address, .amount] | @tsv' "${FIX}")

echo "==> 4. Recovery: CW20 + native drained from RecoveryFromAddress"
while IFS= read -r c; do
  LEFT=$(smart "${c}" "{\"balance\":{\"address\":\"${RECOVERY_FROM}\"}}" | jq -r '.data.balance // "?"')
  GOT=$(smart "${c}" "{\"balance\":{\"address\":\"${RECOVERY_ADDR}\"}}" | jq -r '.data.balance // "?"')
  [[ "${LEFT}" == "0" && "${GOT}" != "0" && "${GOT}" != "?" ]] \
    && pass "cw20 ${c}: from=0 recovery=${GOT}" \
    || fail "cw20 ${c}: from=${LEFT} recovery=${GOT}"
done < <(jq -r '.recovery_assets[]' "${FIX}")

while IFS= read -r d; do
  LEFT=$(bal "${RECOVERY_FROM}" "${d}")
  GOT=$(bal "${RECOVERY_ADDR}" "${d}")
  [[ "${LEFT}" == "0" && "${GOT}" != "0" ]] \
    && pass "native ${d}: from=0 recovery=${GOT}" \
    || fail "native ${d}: from=${LEFT} recovery=${GOT}"
done < <(jq -r '.recovery_native_denoms[]' "${FIX}")

echo "==> 5. Pools paused"
POOL_V2=$(jq -r '.pause_pool_v2' "${FIX}")
POOL_V3=$(jq -r '.pause_pool_v3' "${FIX}")
ADMIN=$(jq -r '.admin_contract' "${FIX}")
WL=$(smart "${POOL_V2}" "{\"trader_is_whitelisted\":{\"trader\":\"${ADMIN}\"}}" | jq -r '.data')
[[ "${WL}" == "false" ]] && pass "pair whitelist enabled (admin no longer open)" \
  || fail "pair trader_is_whitelisted=${WL}, expected false"
PAUSED=$(smart "${POOL_V3}" '{"is_paused":{}}' | jq -r '.data')
[[ "${PAUSED}" == "true" ]] && pass "oraiswap-v3 paused" || fail "v3 is_paused=${PAUSED}, expected true"

echo "==> 6. Blacklist enforcement wiring"
# SCOPE NOTE: the outbound-send restriction CANNOT be proven end-to-end with production
# fixtures. Every BlacklistAddresses entry is a real mainnet account whose key nobody holds,
# x/txfees exposes no msg to blacklist an address we DO control, and no query to read the
# set back. A "tx from a blacklisted account failed" result here could only ever mean
# "key not found" -- a vacuous pass. So this step asserts the two things that ARE
# observable, and the enforcement itself is covered by the Go suite:
#   app/upgrades/v05014 :: TestBlacklistRestrictionOnlyAfterForkBlock
#     -> outbound blocked at ForkHeight+1, inbound allowed, funds stay frozen.
BLK=$(jq -r '.blacklist_addresses[0]' "${FIX}")
EXPECTED_BL=$(jq -r '.blacklist_addresses | length' "${FIX}")

# (a) the fork populated the txfees store that the bank SendRestriction reads
BLLOG=$(grep -c "txfees blacklist enabled" "${HOME_A}/start.log" || true)
# Count the per-address "added" lines rather than parsing count=N: the log is ANSI-coloured,
# so an escape sequence sits between "count=" and the digits.
GOT=$(grep -c "txfees blacklist: added" "${HOME_A}/start.log" || true)
if [[ "${BLLOG}" -gt 0 ]]; then
  [[ "${GOT}" == "${EXPECTED_BL}" ]] \
    && pass "fork wrote ${GOT} blacklist entries into the txfees store" \
    || fail "fork added ${GOT} blacklist entries, expected ${EXPECTED_BL}"
else
  fail "no 'txfees blacklist enabled' line -- enableTxFeesBlacklist did not run"
fi

# (b) inbound stays allowed: funds land on a blacklisted account and then freeze there
otx bank send tester "${BLK}" "5000000${DENOM}" >/dev/null 2>&1 || true
sleep 6
FUNDED=$(bal "${BLK}")
[[ "${FUNDED}" != "0" ]] \
  && pass "inbound to blacklist allowed, now frozen (${FUNDED}) -- no key exists to move it" \
  || fail "inbound send to a blacklisted account did not land"

echo "==> 7. Normal (non-blacklisted) transfers still work"
BEFORE=$(bal "${ADDR_A}")
otx bank send tester "${ADDR_A}" "2000000${DENOM}" >/dev/null 2>&1 || true
sleep 6
AFTER=$(bal "${ADDR_A}")
[[ "${AFTER}" -gt "${BEFORE}" ]] && pass "tester -> node-a ok (${BEFORE} -> ${AFTER})" \
  || fail "normal transfer did not land (${BEFORE} -> ${AFTER})"

echo "==> 8. EVM JSON-RPC must be dead (app.toml has enable = true)"
grep -A3 '^\[json-rpc\]' "${DATA_DIR}/node-a/config/app.toml" | grep -q '^enable = true' \
  && pass "app.toml still requests json-rpc (kill switch genuinely under test)" \
  || fail "app.toml does not request json-rpc — test is vacuous"
if curl -sf -m 5 -X POST "${JSONRPC_A}" \
     -H 'Content-Type: application/json' \
     -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' >/dev/null 2>&1; then
  fail "eth_blockNumber answered — JSON-RPC is live"
else
  pass "no JSON-RPC listener (eth_* unreachable)"
fi

echo "==> 9. node-b (stale v0.50.13b) must have diverged"
BLOG=$(tail -400 "$(node_home b)/start.log" 2>/dev/null || true)
if echo "${BLOG}" | grep -qiE "wrong Block.Header.AppHash|apphash|app hash|panic"; then
  pass "node-b reports apphash/consensus failure"
  echo "${BLOG}" | grep -iE "wrong Block.Header.AppHash|apphash|app hash" | tail -2 | sed 's/^/      /'
else
  HB=$(height b)
  if [[ "${HB}" -gt "${FORK_HEIGHT}" ]]; then
    fail "node-b followed past the fork on the OLD binary (height ${HB}) — divergence not enforced"
  else
    pass "node-b stuck at height ${HB} (did not follow past fork ${FORK_HEIGHT})"
  fi
fi

echo
echo "==> Summary"
if [[ "${FAILURES}" -eq 0 ]]; then
  echo "✓ all production-fixture fork checks passed"
else
  echo "✗ ${FAILURES} check(s) failed"
  exit 1
fi
