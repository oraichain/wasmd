#!/bin/bash

set -ux

WASM_TOKEN_FACTORY=${WASM_TOKEN_FACTORY:-"$PWD/scripts/wasm_file/tokenfactory-no-restrict.wasm"}
ARGS="--from gnad --home .oraid --chain-id testing -y --keyring-backend test --fees 200orai --gas auto --gas-adjustment 1.5"

# Deploy new mock oraidex contract
store_ret=$(oraid tx wasm store $WASM_TOKEN_FACTORY $ARGS --output json)
tx_hash=$(echo $store_ret | jq -r '.txhash')

# Wait for transaction to be committed
sleep 2

# Get code_id using tx_hash
code_id=$(oraid query tx $tx_hash --output json | jq -r '.events[] | select(.type == "store_code") | .attributes[] | select(.key == "code_id") | .value')

# Instantiate contract
INSTANTIATE_MSG='{}'
instantiate_ret=$(oraid tx wasm instantiate $code_id "$INSTANTIATE_MSG" --label 'testing' --admin $(oraid keys show gnad --keyring-backend test --home .oraid -a) $ARGS --output json)
instantiate_tx_hash=$(echo $instantiate_ret | jq -r '.txhash')

# Wait for transaction to be committed
sleep 2

# Get contract address
token_factoyr_address=$(oraid query tx $instantiate_tx_hash --output json | jq -r '.events[] | select(.type == "instantiate") | .attributes[] | select(.key == "_contract_address") | .value')
sleep 2

echo $token_factoyr_address