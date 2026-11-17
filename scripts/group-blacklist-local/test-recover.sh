#!/usr/bin/env bash
# End-to-end check of MsgRemoveBlacklistAndRecover through the foundation group policy.
# Requires: local testnet running, validator1 is a member of group 1 (threshold 1).
# Usage: bash scripts/group-blacklist-local/test-recover.sh
set -euo pipefail

source "$(dirname "$0")/env.sh"

TX_FLAGS=(--from "$FROM" --home "$HOME1" --keyring-backend test --chain-id "$CHAIN_ID"
  --node "$NODE" --gas-prices 0.001orai -y -o json)
TMP="$(mktemp -d)"
FAILED=0

pass() { echo "  PASS: $*"; }
fail() { echo "  FAIL: $*"; FAILED=1; }

key_addr() {
  local name=$1
  if ! oraid keys show "$name" -a --home "$HOME1" --keyring-backend test >/dev/null 2>&1; then
    oraid keys add "$name" --home "$HOME1" --keyring-backend test >/dev/null 2>&1
  fi
  oraid keys show "$name" -a --home "$HOME1" --keyring-backend test
}

balance() {
  oraid q bank balance "$1" orai --node "$NODE" -o json |
    python3 -c 'import json,sys; d=json.load(sys.stdin); print((d.get("balance") or d)["amount"])'
}

blacklisted() {
  oraid q txfees check-blacklist "$1" --node "$NODE" -o json |
    python3 -c 'import json,sys; print(str(json.load(sys.stdin).get("blacklisted", False)).lower())'
}

# Prints "<code> <raw_log>" of a committed tx; waits for inclusion.
wait_tx() {
  local hash=$1
  for _ in $(seq 1 20); do
    if out=$(oraid q tx --type=hash "$hash" --node "$NODE" -o json 2>/dev/null); then
      echo "$out" > "$TMP/last-tx.json"
      python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); print(d["code"], d["raw_log"][:300])' "$TMP/last-tx.json"
      return
    fi
    sleep 1
  done
  echo "timeout waiting for $hash"; return 1
}

# Broadcast a tx; prints "<code> <raw_log>" (CheckTx rejection or committed result).
run_tx() {
  local res
  res=$(oraid tx "$@" 2>/dev/null) || true
  local code hash
  code=$(echo "$res" | python3 -c 'import json,sys; print(json.load(sys.stdin)["code"])' 2>/dev/null || echo "?")
  hash=$(echo "$res" | python3 -c 'import json,sys; print(json.load(sys.stdin)["txhash"])' 2>/dev/null || echo "")
  if [[ "$code" != "0" ]]; then
    echo "$code $(echo "$res" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("raw_log","")[:300])' 2>/dev/null || echo "$res" | head -c 300)"
    return
  fi
  wait_tx "$hash"
}

# Submits a proposal with one message and EXEC_TRY; prints the EventExec result.
propose() {
  local msg_json=$1 title=$2
  cat > "$TMP/proposal.json" <<EOF
{
  "group_policy_address": "$POLICY_ADDR",
  "messages": [$msg_json],
  "metadata": "",
  "title": "$title",
  "summary": "$title",
  "proposers": ["$ADMIN"]
}
EOF
  local r
  r=$(run_tx group submit-proposal "$TMP/proposal.json" --exec try --gas 800000 "${TX_FLAGS[@]}")
  if [[ "${r%% *}" != "0" ]]; then echo "SUBMIT_FAILED $r"; return; fi
  python3 - "$TMP/last-tx.json" <<'PY'
import json, sys
d = json.load(open(sys.argv[1]))
for e in d["events"]:
    if e["type"] == "cosmos.group.v1.EventExec":
        a = {x["key"]: json.loads(x["value"]) if x["value"].startswith('"') else x["value"] for x in e["attributes"]}
        print(a.get("result", ""), a.get("logs", ""))
        break
else:
    print("NO_EXEC_EVENT")
PY
}

VICTIM=$(key_addr recover-victim)
DEST=$(key_addr recover-dest)
echo "victim: $VICTIM"
echo "dest:   $DEST"

echo "== 1. fund victim"
r=$(run_tx bank send "$FROM" "$VICTIM" 5000000orai "${TX_FLAGS[@]}")
[[ "${r%% *}" == "0" ]] && pass "funded victim" || fail "fund victim: $r"

if [[ "$(blacklisted "$VICTIM")" != "true" ]]; then
  echo "== 2. blacklist victim via group proposal"
  r=$(propose "{\"@type\":\"/cosmwasm.txfees.v1.MsgAddBlacklist\",\"authority\":\"$POLICY_ADDR\",\"address\":\"$VICTIM\"}" "test: blacklist victim")
  [[ "$r" == PROPOSAL_EXECUTOR_RESULT_SUCCESS* ]] && pass "add blacklist executed" || fail "add blacklist: $r"
fi
[[ "$(blacklisted "$VICTIM")" == "true" ]] && pass "victim is blacklisted" || fail "victim not blacklisted"

echo "== 3. victim cannot send out"
r=$(run_tx bank send "$VICTIM" "$DEST" 1orai --from recover-victim --home "$HOME1" --keyring-backend test \
  --chain-id "$CHAIN_ID" --node "$NODE" --gas-prices 0.001orai -y -o json)
[[ "${r%% *}" != "0" ]] && pass "outbound send rejected (${r:0:120})" || fail "blacklisted victim could send: $r"

echo "== 4. victim can still receive"
before=$(balance "$VICTIM")
r=$(run_tx bank send "$FROM" "$VICTIM" 1000000orai "${TX_FLAGS[@]}")
after=$(balance "$VICTIM")
[[ "${r%% *}" == "0" && $((after - before)) -eq 1000000 ]] && pass "inbound transfer received" || fail "inbound: $r ($before -> $after)"

echo "== 5. RemoveBlacklistAndRecover victim -> dest"
victim_before=$(balance "$VICTIM")
dest_before=$(balance "$DEST")
echo "  victim=$victim_before dest=$dest_before"
r=$(propose "{\"@type\":\"/cosmwasm.txfees.v1.MsgRemoveBlacklistAndRecover\",\"authority\":\"$POLICY_ADDR\",\"address\":\"$VICTIM\",\"to_address\":\"$DEST\"}" "test: recover victim")
[[ "$r" == PROPOSAL_EXECUTOR_RESULT_SUCCESS* ]] && pass "recover executed" || fail "recover: $r"
victim_after=$(balance "$VICTIM")
dest_after=$(balance "$DEST")
echo "  victim=$victim_after dest=$dest_after"
[[ "$victim_after" == "0" ]] && pass "victim drained to 0" || fail "victim still has $victim_after"
[[ $((dest_after - dest_before)) -eq $victim_before ]] && pass "dest received $victim_before" || fail "dest delta $((dest_after - dest_before)) != $victim_before"
[[ "$(blacklisted "$VICTIM")" == "false" ]] && pass "victim removed from blacklist" || fail "victim still blacklisted"

echo "== 6. recover again on a non-blacklisted address must fail"
r=$(propose "{\"@type\":\"/cosmwasm.txfees.v1.MsgRemoveBlacklistAndRecover\",\"authority\":\"$POLICY_ADDR\",\"address\":\"$VICTIM\",\"to_address\":\"$DEST\"}" "test: recover non-blacklisted")
[[ "$r" == PROPOSAL_EXECUTOR_RESULT_FAILURE* ]] && pass "rejected (${r:0:160})" || fail "expected failure, got: $r"

rm -rf "$TMP"
echo
[[ $FAILED -eq 0 ]] && echo "ALL CHECKS PASSED" || { echo "SOME CHECKS FAILED"; exit 1; }
