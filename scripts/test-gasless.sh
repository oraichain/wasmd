#!/bin/bash

set -ux

CHAIN_ID=${CHAIN_ID:-testing}
USER=${USER:-tupt}
WASM_PATH=${WASM_PATH:-"$PWD/scripts/wasm_file/counter_high_gas_cost.wasm"}
EXECUTE_MSG=${EXECUTE_MSG:-'{"ping":{}}'}
NODE_HOME=${NODE_HOME:-"$PWD/.oraid"}
ARGS="--from $USER --chain-id $CHAIN_ID -y --keyring-backend test --gas auto --gas-adjustment 1.5 -b sync --home $NODE_HOME"
HIDE_LOGS="/dev/null"
PROPOSAL_TEMPLATE_PATH="$PWD/scripts/json/set-gasless-proposal.json"

# prepare a new contract for gasless
store_code_txhash=$(oraid tx wasm store $WASM_PATH $ARGS --output json | jq -r '.txhash')
# sleep 2s for tx already in block
sleep 2
code_id=$(oraid query tx $store_code_txhash --output json | jq -r '.events[4].attributes[] | select(.key | contains("code_id")).value')

VALIDATOR_ADDRESS=$(oraid keys show $USER --keyring-backend test --home $NODE_HOME -a)
oraid tx wasm instantiate $code_id '{}' --label 'testing' --admin $VALIDATOR_ADDRESS $ARGS >$HIDE_LOGS
# sleep 2s for tx already in block
sleep 2
contract_address=$(oraid query wasm list-contract-by-code $code_id --output json | jq '.contracts[0]' | sed 's/^"//;s/"$//')

# try executing something
exec_before_txhash=$(oraid tx wasm execute $contract_address $EXECUTE_MSG $ARGS --output json | jq -r '.txhash')
# sleep 20s for tx already in block
sleep 20
gas_used_before=$(oraid query tx $exec_before_txhash --output json | jq -r '.gas_used | tonumber')
echo "gas used before gasless: $gas_used_before"

# set gasless proposal
AUTHORITY_ADRESS=$(oraid query auth module-account gov --output json | jq '.account.value.address' | sed 's/^"//;s/"$//')
SET_GASLESS_ARGS="--title "set_gasless" --summary "set_gasless" --deposit 10000000orai --authority $AUTHORITY_ADRESS $ARGS"
set_gasless_txhash=$(oraid tx wasm submit-proposal set-gasless $contract_address $SET_GASLESS_ARGS --output json | jq -r '.txhash')
sleep 2
proposal_id=$(oraid query tx $set_gasless_txhash --output json | jq -r '.events[4].attributes[] | select(.key | contains("proposal_id")).value')
oraid tx gov vote $proposal_id yes $ARGS >$HIDE_LOGS

# check if proposal pass or not
sleep 5
proposal_status=$(oraid query gov proposal $proposal_id --output json | jq .proposal.status)
if [ $proposal_status -eq "4" ]; then
    echo "The proposal has failed"
    exit 1
fi
if [ $proposal_status -ne "3" ]; then
    echo "The proposal has not passed yet"
    exit 1
fi

# try executing again after set gas less
exec_after_txhash=$(oraid tx wasm execute $contract_address $EXECUTE_MSG $ARGS --output json | jq -r '.txhash')
# sleep 20s for tx already in block
sleep 20
gas_used_after=$(oraid query tx $exec_after_txhash --output json | jq -r '.gas_used | tonumber')
echo "gas used after set gasless: $gas_used_after"

echo "Test gasless passed!"
