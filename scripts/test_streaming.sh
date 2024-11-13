#!/bin/bash
set -ux

CHAINID=${CHAINID:-testing}
USER=${USER:-tupt}
MONIKER=${MONIKER:-node001}
HIDE_LOGS="/dev/null"
WASM_PATH=${WASM_PATH:-"$PWD/scripts/wasm_file/tokenfactory.wasm"}
NODE_HOME="$PWD/.oraid"
ARGS="--keyring-backend test --home $NODE_HOME"
BANKTX_ARGS="$ARGS --chain-id $CHAINID --gas 200000 --fees 2orai --node http://localhost:26657 --yes"
WASMTX_ARGS="$ARGS --from $USER --chain-id $CHAINID -y --gas auto --gas-adjustment 1.5 -b sync"

# send orai to another address
VALIDATOR_ADDRESS=$(oraid keys show $USER -a --keyring-backend test --home $NODE_HOME)
oraid tx bank send $VALIDATOR_ADDRESS orai1kzkf6gttxqar9yrkxfe34ye4vg5v4m588ew7c9 10000orai $BANKTX_ARGS

sleep 5
# execute wasm tx -> store code
oraid tx wasm store $WASM_PATH $WASMTX_ARGS --output json

sleep 5
# execute wasm tx -> instantiate contract
oraid tx wasm instantiate 1 '{}' --label 'tokenfactory cw bindings testing' --admin $VALIDATOR_ADDRESS $WASMTX_ARGS >$HIDE_LOGS

sleep 2
contract_address=$(oraid query wasm list-contract-by-code 1 --output json | jq -r '.contracts | last')

oraid tx bank send $VALIDATOR_ADDRESS $contract_address 100000000orai $BANKTX_ARGS >$HIDE_LOGS

subdenom="usdc"
CREATE_DENOM_MSG='{"create_denom":{"subdenom":"'"$subdenom"'"}}'
sleep 2
oraid tx wasm execute $contract_address $CREATE_DENOM_MSG $WASMTX_ARGS >$HIDE_LOGS

sleep 5
# re-send orai to another address
oraid tx bank send $VALIDATOR_ADDRESS orai1kzkf6gttxqar9yrkxfe34ye4vg5v4m588ew7c9 10000orai $BANKTX_ARGS
