#!/bin/bash

set -ux

PRICE_QUERY_WASM_PATH=${PRICE_QUERY_WASM_PATH:-"$PWD/scripts/wasm_file/price-query-local.wasm"}
WASM_MOCK_ORAIDEX_PATH=${WASM_MOCK_ORAIDEX_PATH:-"$PWD/scripts/wasm_file/mock-oraidex-v3.wasm"}
CHAIN_ID=${CHAIN_ID:-testing}
USER=${USER:-"validator1"}
NODE_HOME=${NODE_HOME:-"$HOME/.oraid/validator1"}
ARGS="--from $USER --chain-id $CHAIN_ID -y --keyring-backend test --gas auto --gas-adjustment 1.5 -b sync --home $NODE_HOME"
HIDE_LOGS="/dev/null"

# create a token-factory

# deploy mock oraidex contract

# deploy price contract

# add pool

# gov set param

# gov add token fee

# try execute tx with custom fee
