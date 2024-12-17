#!/bin/bash

set -ux

source "$(dirname $0)/../utils.sh"

CHAIN_ID=${CHAIN_ID:-testing}
USER=${USER:-tupt}
USER2=${USER2:-''}
WASM_PATH=${WASM_PATH:-"$PWD/scripts/wasm_file/swapmap.wasm"}
EXECUTE_MSG=${EXECUTE_MSG:-'{"ping":{}}'}
NODE_HOME=${NODE_HOME:-"$HOME/.oraid"}
VALIDATOR1_HOME=${VALIDATOR1_HOME:-"$NODE_HOME/$USER"}
VALIDATOR2_HOME="$NODE_HOME/$USER2"
VALIDATOR1_ARG="--from $USER --chain-id $CHAIN_ID -y --keyring-backend test --gas auto --gas-adjustment 1.5 -b sync --home $VALIDATOR1_HOME"
VALIDATOR2_ARG="--from $USER2 --chain-id $CHAIN_ID -y --keyring-backend test --gas auto --gas-adjustment 1.5 -b sync --home $VALIDATOR2_HOME"
HIDE_LOGS="/dev/null"
PROPOSAL_TEMPLATE_PATH="$PWD/scripts/json/set-gasless-proposal.json"

result_txhash=$(oraid tx wasm execute $contract_address $EXECUTE_MSG $VALIDATOR1_ARG --output json | jq -r '.txhash')
# wait for tx included in a block
sleep 20
gas_used_after=$(oraid query tx $result_txhash --output json | jq -r '.gas_used | tonumber')
echo "gas used after gasless: $gas_used_after"
# 1.9 is a magic number chosen to check that if the gas used after gasless has dropped significantly or not
gas_used_compare=$(echo "$gas_used_before / 1.9 / 1" | bc)
echo "gas_used_compare: $gas_used_compare"
if [[ $gas_used_compare -lt $gas_used_after ]]; then
    echo "Gas used after is not small enough!"
    # exit 1
fi

# try testing with non-gasless contract with the same logic, should have much higher gas
oraid tx wasm instantiate $code_id '{}' --label 'testing2' --admin $(oraid keys show $USER --keyring-backend test --home $VALIDATOR1_HOME -a) $VALIDATOR1_ARG >$HIDE_LOGS
# wait for tx included in a block
sleep 2
non_gasless_contract_address=$(oraid query wasm list-contract-by-code $code_id --output json | jq -r '.contracts[1]')
# non_gasless_contract_address=$(oraid query wasm list-contract-by-code $code_id --output json | jq -r '.contracts[1]')
echo 'non_gasless_contract_address:' $non_gasless_contract_address
result_txhash=$(oraid tx wasm execute $non_gasless_contract_address $EXECUTE_MSG $VALIDATOR1_ARG --output json | jq -r '.txhash')
# wait for tx included in a block
sleep 20
gas_used_non_gasless=$(oraid query tx $result_txhash --output json | jq -r '.gas_used | tonumber')
if [[ $gas_used_non_gasless -le $gas_used_after ]]; then
    echo "Gas used non gas less is not large enough! Contract gasless test failed"
    exit 1
fi

echo "Gasless tests passed!"
