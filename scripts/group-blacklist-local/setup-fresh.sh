#!/usr/bin/env bash
# Fresh local testnet + foundation x/group for txfees blacklist tests.
# Wipes ~/.oraid, starts 3 validators (screen), enables CORS for the FE, funds the FE wallet
# and creates the foundation group (policy is its own admin).
#
# Usage: bash scripts/group-blacklist-local/setup-fresh.sh
#   FE_WALLET=orai1... to fund/add a different Keplr wallet as group member.
set -euo pipefail

REPO="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$REPO"
FE_WALLET="${FE_WALLET:-orai1y38tt420cfv8ad7pv8mq3ltjl2553edp749rcq}"
TMP="$(mktemp -d)"

step() { echo; echo "== $*"; }

wait_blocks() {
  local start="" h
  for _ in $(seq 1 60); do
    h=$(curl -s -m 2 http://127.0.0.1:26657/status 2>/dev/null |
      python3 -c 'import json,sys; print(json.load(sys.stdin)["result"]["sync_info"]["latest_block_height"])' 2>/dev/null || true)
    if [[ -n "$h" ]]; then
      [[ -z "$start" ]] && start=$h
      if (( h > start + 1 )); then echo "  chain at height $h"; return; fi
    fi
    sleep 1
  done
  echo "  chain is not producing blocks"; exit 1
}

# Broadcasts and waits for inclusion; exits on failure.
tx() {
  local res hash code
  res=$(oraid tx "$@" --from validator1 --home "$HOME1" --keyring-backend test --chain-id "$CHAIN_ID" \
    --node "$NODE" --gas-prices 0.001orai -y -o json)
  code=$(echo "$res" | python3 -c 'import json,sys; print(json.load(sys.stdin)["code"])')
  hash=$(echo "$res" | python3 -c 'import json,sys; print(json.load(sys.stdin)["txhash"])')
  if [[ "$code" != "0" ]]; then echo "  tx rejected: $res"; exit 1; fi
  for _ in $(seq 1 20); do
    if out=$(oraid q tx --type=hash "$hash" --node "$NODE" -o json 2>/dev/null); then
      code=$(echo "$out" | python3 -c 'import json,sys; print(json.load(sys.stdin)["code"])')
      if [[ "$code" != "0" ]]; then
        echo "  tx failed: $(echo "$out" | python3 -c 'import json,sys; print(json.load(sys.stdin)["raw_log"])')"; exit 1
      fi
      echo "  tx $hash ok"; return
    fi
    sleep 1
  done
  echo "  timeout waiting for $hash"; exit 1
}

step "1. check oraid code signature (macOS kills processes with an invalid one)"
BIN="$(command -v oraid)"
if ! codesign --verify --strict "$BIN" 2>/dev/null; then
  echo "  invalid signature, re-signing $BIN"
  cp "$BIN" "$BIN.resign"
  codesign -s - -f "$BIN.resign"
  mv -f "$BIN.resign" "$BIN"
fi
oraid version

step "2. start fresh 3-validator testnet"
bash scripts/multinode-local-testnet.sh

source scripts/group-blacklist-local/env.sh

step "3. enable CORS + zero min gas price on validator1 (FE uses its RPC/LCD)"
sed -i '' -E 's|^cors_allowed_origins = .*|cors_allowed_origins = ["*"]|' "$HOME1/config/config.toml"
sed -i '' -E 's|^enabled-unsafe-cors = .*|enabled-unsafe-cors = true|' "$HOME1/config/app.toml"
sed -i '' -E 's|^minimum-gas-prices = .*|minimum-gas-prices = "0orai"|' "$HOME1/config/app.toml"
screen -S validator1 -X quit || true
sleep 2
screen -S validator1 -d -m oraid start --home "$HOME1"
wait_blocks

step "4. fund FE wallet $FE_WALLET"
tx bank send "$ADMIN" "$FE_WALLET" 1000000000orai

step "5. create foundation group (members: validator1 + FE wallet, threshold 1, policy as admin)"
cat > "$TMP/members.json" <<EOF
{
  "members": [
    {"address": "$ADMIN", "weight": "1", "metadata": "validator1"},
    {"address": "$FE_WALLET", "weight": "1", "metadata": "fe-wallet"}
  ]
}
EOF
tx group create-group-with-policy "$ADMIN" "foundation-group" "foundation-policy" \
  "$TMP/members.json" "$DIR/02-decision-policy.json" --group-policy-as-admin --gas 500000

step "6. verify policy address matches x/txfees FoundationGroupPolicyAddress"
created=$(oraid q group group-policies-by-group 1 --node "$NODE" -o json |
  python3 -c 'import json,sys; print(json.load(sys.stdin)["group_policies"][0]["address"])')
admin=$(oraid q group group-info 1 --node "$NODE" -o json |
  python3 -c 'import json,sys; print(json.load(sys.stdin)["info"]["admin"])')
expected=$(sed -n -E 's/^[[:space:]]*FoundationGroupPolicyAddress = "(orai1[a-z0-9]+)"/\1/p' x/txfees/types/keys.go)
echo "  created policy: $created"
echo "  group admin:    $admin"
echo "  keys.go:        $expected"
rm -rf "$TMP"
if [[ "$created" != "$expected" ]]; then
  echo "  MISMATCH: update FoundationGroupPolicyAddress in x/txfees/types/keys.go,"
  echo "  NEXT_PUBLIC_FOUNDATION_GROUP_POLICY in the FE .env.local, then make install + rerun."
  exit 1
fi

echo
echo "Done. validator1=$ADMIN  fe_wallet=$FE_WALLET  policy=$created"
echo "Next: bash scripts/group-blacklist-local/test-recover.sh"
