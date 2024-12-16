#!/bin/bash

set -eu

CHAIN_ID=${CHAIN_ID:-testing}
USER=${USER:-tupt}
NODE_HOME=${NODE_HOME:-"$PWD/.oraid"}
WASM_PATH=${WASM_PATH:-"$PWD/scripts/wasm_file/tokenfactory.wasm"}
ARGS="--from $USER --chain-id $CHAIN_ID -y --keyring-backend test --gas auto --gas-adjustment 1.5 -b sync --home $NODE_HOME"
user_address=$(oraid keys show $USER --keyring-backend test --home $NODE_HOME -a)
HIDE_LOGS="/dev/null"

# deploy cw-bindings contract
store_txhash=$(oraid tx wasm store $WASM_PATH $ARGS --output json | jq -r '.txhash')
# need to sleep 1s
sleep 2
code_id=$(oraid query tx $store_txhash --output json | jq -r '.events[4].attributes[] | select(.key | contains("code_id")).value')
oraid tx wasm instantiate $code_id '{}' --label 'tokenfactory cw bindings testing' --admin $user_address $ARGS >$HIDE_LOGS
sleep 2
contract_address=$(oraid query wasm list-contract-by-code $code_id --output json | jq -r '.contracts[0]')
echo $contract_address

subdenom="usdo"
metadata='{"description":"foobar","name":"'"$subdenom"'","symbol":"'"$subdenom"'","base":"'"factory/$contract_address/$subdenom"'","display":"'"$subdenom"'","denom_units":[{"denom":"'"factory/$contract_address/$subdenom"'","exponent":0,"aliases":["'"$subdenom"'"]},{"denom":"'"$subdenom"'","exponent":6,"aliases":["'"$subdenom"'"]}],"uri":"foobar","uri_hash":"foobar"}'
CREATE_DENOM_MSG='{"create_denom":{"subdenom":"'"$subdenom"'","metadata":'$metadata'}}'
QUERY_DENOM_MSG='{"get_denom":{"creator_address":"'"$user_address"'","subdenom":"'"$subdenom"'"}}'

echo "create denom msg: $CREATE_DENOM_MSG"
echo "query denom msg: $QUERY_DENOM_MSG"

# send to the contract some funds to create denom
oraid tx bank send $user_address $contract_address 100000000orai $ARGS >$HIDE_LOGS

# create denom
# sleep 1s to not miss match account sequence
sleep 2
oraid tx wasm execute $contract_address $CREATE_DENOM_MSG --amount 100000000orai $ARGS >$HIDE_LOGS

# query created denom uri and uri_hash
# sleep 2s for create denom tx already in block
created_denom="factory/$contract_address/$subdenom"
sleep 2
uri=$(oraid query bank denom-metadata $created_denom --output json | jq '.metadata.uri' | tr -d '"')
if ! [[ $uri =~ "foobar" ]]; then
    echo "Tokenfactory set metadata binding tests failed! The created uri does not match with our expected uri"
    exit 1
fi

sleep 2
uri_hash=$(oraid query bank denom-metadata $created_denom --output json | jq '.metadata.uri_hash' | tr -d '"')
if ! [[ $uri_hash =~ "foobar" ]]; then
    echo "Tokenfactory set metadata binding tests failed! The created uri hash does not match with our expected uri hash"
    exit 1
fi

echo "Tokenfactory set metadata binding tests passed!"
