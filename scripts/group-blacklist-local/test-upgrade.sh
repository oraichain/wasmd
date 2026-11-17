#!/usr/bin/env bash
# Upgrade test: single-node chain on OLD_REF -> gov software upgrade v0.50.16 -> working tree binary.
# Runs on separate ports/home so the 3-validator FE testnet keeps running.
#
# Usage: bash scripts/group-blacklist-local/test-upgrade.sh
#   OLD_REF=release/v0.50.x (default) | v0.50.15-rc.3 | any git ref
set -euo pipefail

REPO="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$REPO"

OLD_REF="${OLD_REF:-release/v0.50.x}"
UPGRADE_NAME="v0.50.16"
WORK="${WORK:-/tmp/oraid-upgrade-test}"
NODE_HOME="$WORK/home"
OLD_BIN="$WORK/oraid-old"
NEW_BIN="$WORK/oraid-new"
CHAIN_ID="upgrade-test"
RPC_PORT=36657
NODE="tcp://127.0.0.1:$RPC_PORT"
LCD="http://127.0.0.1:31317"
LOG="$WORK/node.log"
COMPROMISED=(orai1rhza73qeztj0am3qszgxm7p7tfm5m5gdevdaa9 orai1td0lh3398kerk4tfl7gs30acs50tanhl4qqap8)
POLICY_ADDR="orai1afk9zr2hn2jsac63h4hm60vl9z3e5u69gndzf7c99cqge3vzwjzsp4a86q"
FAILED=0
NODE_PID=""

step() { echo; echo "== $*"; }
pass() { echo "  PASS: $*"; }
fail() { echo "  FAIL: $*"; FAILED=1; }
cleanup() { [[ -n "$NODE_PID" ]] && kill "$NODE_PID" 2>/dev/null || true; }
trap cleanup EXIT

LDFLAGS="-X github.com/cosmos/cosmos-sdk/version.Name=orai -X github.com/cosmos/cosmos-sdk/version.AppName=oraid \
-X github.com/CosmWasm/wasmd/app.Bech32Prefix=orai -X github.com/cosmos/cosmos-sdk/version.BuildTags=netgo,ledger"

# go build output can carry an invalid ad-hoc signature on macOS, which gets the process SIGKILLed.
build() {
  local src=$1 out=$2 version=$3
  (cd "$src" && go build -mod=readonly -tags "netgo,ledger" \
    -ldflags "$LDFLAGS -X github.com/cosmos/cosmos-sdk/version.Version=$version" -o "$out.tmp" ./cmd/wasmd)
  codesign -s - -f "$out.tmp" >/dev/null 2>&1 || true
  mv -f "$out.tmp" "$out"
  "$out" version
}

height() {
  curl -s -m 2 "http://127.0.0.1:$RPC_PORT/status" 2>/dev/null |
    python3 -c 'import json,sys; print(json.load(sys.stdin)["result"]["sync_info"]["latest_block_height"])' 2>/dev/null || echo 0
}

wait_height() {
  local target=$1
  for _ in $(seq 1 120); do
    (( $(height) >= target )) && return 0
    sleep 1
  done
  return 1
}

start_node() {
  local bin=$1
  "$bin" start --home "$NODE_HOME" >>"$LOG" 2>&1 &
  NODE_PID=$!
}

stop_node() {
  [[ -n "$NODE_PID" ]] && kill "$NODE_PID" 2>/dev/null || true
  for _ in $(seq 1 30); do kill -0 "$NODE_PID" 2>/dev/null || break; sleep 1; done
  NODE_PID=""
}

# Broadcast with given binary as validator; prints "<code> <raw_log>" after inclusion.
tx() {
  local bin=$1; shift
  local res code hash
  res=$("$bin" tx "$@" --from val --home "$NODE_HOME" --keyring-backend test --chain-id "$CHAIN_ID" \
    --node "$NODE" --gas-prices 0.001orai -y -o json 2>&1) || true
  code=$(echo "$res" | python3 -c 'import json,sys; print(json.load(sys.stdin)["code"])' 2>/dev/null || echo "?")
  if [[ "$code" != "0" ]]; then echo "$code $res" | head -c 400; return; fi
  hash=$(echo "$res" | python3 -c 'import json,sys; print(json.load(sys.stdin)["txhash"])')
  for _ in $(seq 1 30); do
    if out=$("$bin" q tx --type=hash "$hash" --node "$NODE" -o json 2>/dev/null); then
      echo "$out" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d["code"], d["raw_log"][:300])'
      return
    fi
    sleep 1
  done
  echo "timeout $hash"
}

step "1. build binaries (old: $OLD_REF, new: working tree)"
mkdir -p "$WORK"
rm -rf "$WORK/src-old"
git worktree remove --force "$WORK/src-old" 2>/dev/null || true
git worktree prune
git worktree add --detach "$WORK/src-old" "$OLD_REF" >/dev/null
build "$WORK/src-old" "$OLD_BIN" "old-$(git rev-parse --short "$OLD_REF")"
build "$REPO" "$NEW_BIN" "new-feat-blacklist"
git worktree remove --force "$WORK/src-old"

step "2. init single-node chain with old binary"
rm -rf "$NODE_HOME" "$LOG"
"$OLD_BIN" init val --chain-id "$CHAIN_ID" --home "$NODE_HOME" >/dev/null 2>&1
"$OLD_BIN" keys add val --keyring-backend test --home "$NODE_HOME" >/dev/null 2>&1
"$OLD_BIN" keys add user --keyring-backend test --home "$NODE_HOME" >/dev/null 2>&1
VAL=$("$OLD_BIN" keys show val -a --keyring-backend test --home "$NODE_HOME")
USER_ADDR=$("$OLD_BIN" keys show user -a --keyring-backend test --home "$NODE_HOME")
G="$NODE_HOME/config/genesis.json"
jq '.app_state.staking.params.bond_denom="orai"
  | .app_state.crisis.constant_fee.denom="orai"
  | .app_state.gov.params.min_deposit[0].denom="orai"
  | .app_state.gov.params.min_deposit[0].amount="10000000"
  | .app_state.gov.params.expedited_min_deposit[0].denom="orai"
  | .app_state.gov.params.expedited_min_deposit[0].amount="50000000"
  | .app_state.gov.params.voting_period="10s"
  | .app_state.gov.params.expedited_voting_period="5s"
  | .app_state.mint.params.mint_denom="orai"' "$G" > "$G.tmp" && mv "$G.tmp" "$G"
"$OLD_BIN" genesis add-genesis-account "$VAL" 1000000000000orai --home "$NODE_HOME" >/dev/null
"$OLD_BIN" genesis add-genesis-account "$USER_ADDR" 1000000000orai --home "$NODE_HOME" >/dev/null
"$OLD_BIN" genesis gentx val 500000000orai --keyring-backend test --chain-id "$CHAIN_ID" --home "$NODE_HOME" >/dev/null 2>&1
"$OLD_BIN" genesis collect-gentxs --home "$NODE_HOME" >/dev/null 2>&1

CFG="$NODE_HOME/config/config.toml"; APP="$NODE_HOME/config/app.toml"
sed -i '' -E "s|tcp://127.0.0.1:26657|tcp://127.0.0.1:$RPC_PORT|; s|tcp://127.0.0.1:26658|tcp://127.0.0.1:36658|; s|tcp://0.0.0.0:26656|tcp://0.0.0.0:36656|; s|^pprof_laddr = .*|pprof_laddr = \"\"|" "$CFG"
sed -i '' -E "s|tcp://localhost:1317|tcp://localhost:31317|; s|localhost:9090|localhost:39090|; s|127.0.0.1:8545|127.0.0.1:38545|; s|127.0.0.1:8546|127.0.0.1:38546|; s|127.0.0.1:6065|127.0.0.1:36065|; s|^minimum-gas-prices = .*|minimum-gas-prices = \"0orai\"|" "$APP"

start_node "$OLD_BIN"
wait_height 3 || { echo "old node did not start"; tail -30 "$LOG"; exit 1; }
pass "old node producing blocks"

step "3. pre-upgrade state (old binary)"
r=$(tx "$OLD_BIN" bank send "$VAL" "$USER_ADDR" 12345orai)
[[ "${r%% *}" == "0" ]] && pass "bank send" || fail "bank send: $r"
cat > "$WORK/members.json" <<EOF
{"members":[{"address":"$VAL","weight":"1","metadata":"val"}]}
EOF
cat > "$WORK/policy.json" <<EOF
{"@type":"/cosmos.group.v1.ThresholdDecisionPolicy","threshold":"1","windows":{"voting_period":"60s","min_execution_period":"0s"}}
EOF
r=$(tx "$OLD_BIN" group create-group-with-policy "$VAL" foundation-group foundation-policy \
  "$WORK/members.json" "$WORK/policy.json" --group-policy-as-admin --gas 500000)
[[ "${r%% *}" == "0" ]] && pass "foundation group created" || fail "create group: $r"
created=$("$OLD_BIN" q group group-policies-by-group 1 --node "$NODE" -o json |
  python3 -c 'import json,sys; print(json.load(sys.stdin)["group_policies"][0]["address"])')
[[ "$created" == "$POLICY_ADDR" ]] && pass "policy = FoundationGroupPolicyAddress" || fail "policy $created != $POLICY_ADDR"
user_balance_before=$("$OLD_BIN" q bank balance "$USER_ADDR" orai --node "$NODE" -o json |
  python3 -c 'import json,sys; d=json.load(sys.stdin); print((d.get("balance") or d)["amount"])')

step "4. gov software upgrade proposal $UPGRADE_NAME"
GOV=$("$OLD_BIN" q auth module-account gov --node "$NODE" -o json |
  python3 -c 'import json,sys; d=json.load(sys.stdin)["account"]; print(d.get("value",d).get("address") or d["base_account"]["address"])')
UPGRADE_HEIGHT=$(( $(height) + 40 ))
cat > "$WORK/upgrade-proposal.json" <<EOF
{
  "messages": [{
    "@type": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
    "authority": "$GOV",
    "plan": {"name": "$UPGRADE_NAME", "height": "$UPGRADE_HEIGHT", "info": "upgrade test"}
  }],
  "metadata": "",
  "deposit": "10000000orai",
  "title": "Upgrade to $UPGRADE_NAME",
  "summary": "upgrade test",
  "expedited": false
}
EOF
r=$(tx "$OLD_BIN" gov submit-proposal "$WORK/upgrade-proposal.json" --gas 500000)
[[ "${r%% *}" == "0" ]] && pass "proposal submitted (upgrade height $UPGRADE_HEIGHT)" || { fail "submit: $r"; exit 1; }
r=$(tx "$OLD_BIN" gov vote 1 yes)
[[ "${r%% *}" == "0" ]] && pass "voted yes" || fail "vote: $r"

echo "  waiting for chain to halt at $UPGRADE_HEIGHT..."
for _ in $(seq 1 120); do
  grep -q "UPGRADE \"$UPGRADE_NAME\" NEEDED" "$LOG" && break
  sleep 1
done
grep -q "UPGRADE \"$UPGRADE_NAME\" NEEDED" "$LOG" && pass "old binary halted: upgrade needed" || { fail "no halt"; tail -20 "$LOG"; exit 1; }
stop_node

step "5. start new binary"
start_node "$NEW_BIN"
if wait_height $((UPGRADE_HEIGHT + 3)); then
  pass "new binary producing blocks past $UPGRADE_HEIGHT"
else
  fail "new binary did not progress"; tail -40 "$LOG"; exit 1
fi
applied=$(curl -s "$LCD/cosmos/upgrade/v1beta1/applied_plan/$UPGRADE_NAME" | python3 -c 'import json,sys; print(json.load(sys.stdin)["height"])')
[[ "$applied" == "$UPGRADE_HEIGHT" ]] && pass "applied_plan $UPGRADE_NAME = $applied" || fail "applied_plan = $applied"
grep -q "compromised blacklist enabled" "$LOG" && pass "upgrade handler ran" || fail "upgrade handler log missing"

step "6. post-upgrade checks"
for a in "${COMPROMISED[@]}"; do
  b=$("$NEW_BIN" q txfees check-blacklist "$a" --node "$NODE" -o json | python3 -c 'import json,sys; print(json.load(sys.stdin).get("blacklisted", False))')
  [[ "$b" == "True" ]] && pass "$a blacklisted" || fail "$a not blacklisted ($b)"
done
user_balance_after=$("$NEW_BIN" q bank balance "$USER_ADDR" orai --node "$NODE" -o json |
  python3 -c 'import json,sys; d=json.load(sys.stdin); print((d.get("balance") or d)["amount"])')
[[ "$user_balance_after" == "$user_balance_before" ]] && pass "balances preserved" || fail "balance $user_balance_before -> $user_balance_after"
admin=$("$NEW_BIN" q group group-info 1 --node "$NODE" -o json | python3 -c 'import json,sys; print(json.load(sys.stdin)["info"]["admin"])')
[[ "$admin" == "$POLICY_ADDR" ]] && pass "group state preserved" || fail "group admin $admin"
r=$(tx "$NEW_BIN" bank send "$VAL" "$USER_ADDR" 1orai)
[[ "${r%% *}" == "0" ]] && pass "txs work after upgrade" || fail "post-upgrade send: $r"

step "7. restart new binary (store loading after upgrade)"
stop_node
h=$(( $(height) ))
start_node "$NEW_BIN"
if wait_height $((h + 3)); then pass "restart ok"; else fail "restart failed"; tail -40 "$LOG"; fi

step "8. group blacklist / recover flow on upgraded chain"
if HOME1="$NODE_HOME" NODE="$NODE" CHAIN_ID="$CHAIN_ID" FROM=val \
  bash "$REPO/scripts/group-blacklist-local/test-recover.sh"; then
  pass "test-recover.sh"
else
  fail "test-recover.sh"
fi

echo
echo "logs: $LOG"
[[ $FAILED -eq 0 ]] && echo "UPGRADE TEST PASSED" || { echo "UPGRADE TEST FAILED"; exit 1; }
