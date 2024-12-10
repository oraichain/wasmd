#!/bin/bash

set -ux

CHAIN_ID=${CHAIN_ID:-testing}
USER=${USER:-"validator1"}
NODE_HOME=${NODE_HOME:-"$PWD/.oraid"}
ARGS="--from $USER --chain-id $CHAIN_ID -y --keyring-backend test --gas auto --gas-adjustment 1.5 -b sync --home $NODE_HOME"
VALIDATOR2_ARGS="--from validator2 --chain-id $CHAIN_ID -y --keyring-backend test --gas auto --gas-adjustment 1.5 -b sync --home $HOME/.oraid/validator2"
HIDE_LOGS="/dev/null"

DEFAULT_AMOUNT=100000000
CHANGE_AMOUNT=1000
CHANGE_KEY="DenomCreationFee"
CHANGE_VALUE="[{\"amount\":\"$CHANGE_AMOUNT\",\"denom\":\"orai\"}]"
PROPOSAL_FILE=${PROPOSAL_FILE:-"$PWD/scripts/json/tokenfactory-proposal.json"}

fee_params_denom=$(oraid query tokenfactory params --output json | jq '.params.denom_creation_fee[0].denom')
if ! [[ $fee_params_denom =~ "orai" ]]; then
    echo "Tokenfactory change params tests failed. The tokenfactory fee params denom is not orai"
    exit 1
fi

fee_params_amount=$(oraid query tokenfactory params --output json | jq '.params.denom_creation_fee[0].amount | tonumber')
if ! [[ $fee_params_amount =~ $DEFAULT_AMOUNT ]]; then
    echo "Tokenfactory change params tests failed. The tokenfactory fee params amount is not equal default amount"
    exit 1
fi

update_proposal() {
    cat $PROPOSAL_FILE | jq "$1" >$PWD/scripts/json/temp_proposal.json && mv $PWD/scripts/json/temp_proposal.json $PROPOSAL_FILE
}

# update amount proposal.json
update_proposal ".changes[0].key=$CHANGE_KEY"
update_proposal ".changes[0].value=$CHANGE_VALUE"

store_ret=$(oraid tx tokenfactory param-change $PROPOSAL_FILE $ARGS --output json)
txhash=$(echo $store_ret | jq -r '.txhash')
# sleep 2s before vote to wait for tx confirm
sleep 2
proposal_id=$(oraid query tx $txhash --output json | jq -r '.events[4].attributes[] | select(.key | contains("proposal_id")).value')
oraid tx gov vote $proposal_id yes $ARGS > $HIDE_LOGS && oraid tx gov vote $proposal_id yes $VALIDATOR2_ARGS > $HIDE_LOGS

# sleep to wait til the proposal passes
echo "Sleep til the proposal passes..."
sleep 10

echo "Query new creation amount"
fee_params_amount=$(oraid query tokenfactory params --output json | jq '.params.denom_creation_fee[0].amount | tonumber')
if ! [[ $fee_params_amount =~ $CHANGE_AMOUNT ]]; then
    echo "Tokenfactory change params tests failed. The tokenfactory fee params change proposal is not passed"
    exit 1
fi

echo "Token factory change param proposal test passed"
