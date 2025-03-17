#!/bin/bash

set -ux

PRICE_QUERY_WASM_PATH=${PRICE_QUERY_WASM_PATH:-"$PWD/scripts/wasm_file/price-query-local.wasm"}
WASM_MOCK_ORAIDEX_PATH=${WASM_MOCK_ORAIDEX_PATH:-"$PWD/scripts/wasm_file/mock-oraidex-v3.wasm"}
CHAIN_ID=${CHAIN_ID:-testing}
USER=${USER:-"validator1"}
NODE_HOME=${NODE_HOME:-"$PWD/.oraid"}
ARGS="--from $USER --home $NODE_HOME --keyring-backend test --chain-id $CHAIN_ID -y --gas auto --gas-adjustment 1.5 --fees 1000orai -b sync"
CREATE_ARGS="--from $USER --home $NODE_HOME --keyring-backend test --chain-id $CHAIN_ID -y --gas auto --gas-adjustment 1.5 --fees 10000000orai -b sync"
HIDE_LOGS="/dev/null"
PARAMS_PROPOSAL_FILE=${PARAMS_PROPOSAL_FILE:-"$PWD/scripts/json/txfees-params-proposal.json"}
ADD_FEE_PROPOSAL_FILE=${ADD_FEE_PROPOSAL_FILE:-"$PWD/scripts/json/txfees-add-fee-proposal.json"}

##
## create a token-factory
##
fee_params=$(oraid query tokenfactory params --output json | jq '.params.denom_creation_fee[0].denom')
if ! [[ $fee_params =~ "orai" ]]; then
    echo "Txfees tests failed. The tokenfactory fee params is not orai"
    exit 1
fi

# try creating a new denom
denom_name="usdai"
oraid tx tokenfactory create-denom $denom_name $CREATE_ARGS >$HIDE_LOGS
sleep 2

user_address=$(oraid keys show $USER --home $NODE_HOME --keyring-backend test -a)
full_denom_name="factory/$user_address/$denom_name"

admin=$(oraid query tokenfactory denom-authority-metadata $full_denom_name --output json | jq '.authority_metadata.admin')
echo "admin: $admin"
if ! [[ $admin =~ $user_address ]]; then
    echo "Txfees tests failed. The tokenfactory admin does not match the creator"
    exit 1
fi

# try mint
oraid tx tokenfactory mint 1000000000000000$full_denom_name $ARGS >$HIDE_LOGS

# query balance after mint
# need sleep 2s
sleep 2
tokenfactory_balance=$(oraid query bank balance $user_address $full_denom_name --output json | jq '.balance.amount | tonumber')
if [[ $tokenfactory_balance -ne 1000000000000000 ]]; then
    echo "Txfees tests failed. The tokenfactory balance does not increase after mint"
    exit 1
fi

##
## deploy mock oraidex contract
##
store_ret=$(oraid tx wasm store $WASM_MOCK_ORAIDEX_PATH $ARGS --output json)
tx_hash=$(echo $store_ret | jq -r '.txhash')
sleep 2

code_id=$(oraid query tx $tx_hash --output json | jq -r '.events[] | select(.type == "store_code") | .attributes[] | select(.key == "code_id") | .value')

INSTANTIATE_MSG='{}'
instantiate_ret=$(oraid tx wasm instantiate $code_id "$INSTANTIATE_MSG" --label 'testing' --admin $user_address $ARGS --output json)
instantiate_tx_hash=$(echo $instantiate_ret | jq -r '.txhash')

sleep 2
mock_oraidex_address=$(oraid query tx $instantiate_tx_hash --output json | jq -r '.events[] | select(.type == "instantiate") | .attributes[] | select(.key == "_contract_address") | .value')
echo "mock_oraidex_address: $mock_oraidex_address"

##
## deploy price contract
##
store_ret=$(oraid tx wasm store $PRICE_QUERY_WASM_PATH $ARGS --output json)
tx_hash=$(echo $store_ret | jq -r '.txhash')

sleep 2
code_id=$(oraid query tx $tx_hash --output json | jq -r '.events[] | select(.type == "store_code") | .attributes[] | select(.key == "code_id") | .value')

INSTANTIATE_MSG=$(jq -n --arg base_token "orai" --arg oraidex_v3_addr "$mock_oraidex_address" \
    '{"base_token": $base_token, "oraidex_v3_addr": $oraidex_v3_addr}')
instantiate_ret=$(oraid tx wasm instantiate $code_id "$INSTANTIATE_MSG" --label 'testing' --admin $user_address $ARGS --output json)
instantiate_tx_hash=$(echo $instantiate_ret | jq -r '.txhash')

sleep 2
price_contract_address=$(oraid query tx $instantiate_tx_hash --output json | jq -r '.events[] | select(.type == "instantiate") | .attributes[] | select(.key == "_contract_address") | .value')

##
## add pool
##
ADD_POOL=$(
    jq -n --arg quote_token "$full_denom_name" \
        '{"add_pool": {"quote_token": $quote_token,"fee_tier": {"fee": 3000000000, "tick_spacing": 100}}}'
)
oraid tx wasm execute $price_contract_address "$ADD_POOL" $ARGS --output json
sleep 2

##
## gov set param
##
update_proposal() {
    cat $PARAMS_PROPOSAL_FILE | jq "$1" >$PWD/scripts/json/temp_proposal.json && mv $PWD/scripts/json/temp_proposal.json $PARAMS_PROPOSAL_FILE
}

update_proposal ".messages[0][\"params\"][\"price_contract_address\"]=\"$price_contract_address\""

tx_hash=$(oraid tx gov submit-proposal $PARAMS_PROPOSAL_FILE $ARGS --output json | jq -r '.txhash')
sleep 2

proposal_id=$(oraid query tx $tx_hash --output json | jq -r '.events[] | select(.type == "proposal_deposit") | .attributes[] | select(.key == "proposal_id") | .value')
oraid tx gov vote $proposal_id yes $ARGS >$HIDE_LOGS
sleep 30

query_price_contract=$(oraid query txfees params --output json | jq -r '.params.price_contract_address')
if ! [[ $query_price_contract =~ $price_contract_address ]]; then
    echo "Txfees tests failed. Can not set price contract params"
    exit 1
fi

##
## gov add token fee
##
update_proposal() {
    cat $ADD_FEE_PROPOSAL_FILE | jq "$1" >$PWD/scripts/json/temp_proposal.json && mv $PWD/scripts/json/temp_proposal.json $ADD_FEE_PROPOSAL_FILE
}

update_proposal ".messages[0][\"config\"][\"denom\"]=\"$full_denom_name\""

tx_hash=$(oraid tx gov submit-proposal $ADD_FEE_PROPOSAL_FILE $ARGS --output json | jq -r '.txhash')
sleep 2

proposal_id=$(oraid query tx $tx_hash --output json | jq -r '.events[] | select(.type == "proposal_deposit") | .attributes[] | select(.key == "proposal_id") | .value')
oraid tx gov vote $proposal_id yes $ARGS >$HIDE_LOGS
sleep 30

query_denom_config=$(oraid query txfees token-config $full_denom_name --output json | jq -r '.config.denom')
if ! [[ $query_denom_config =~ $full_denom_name ]]; then
    echo "Txfees tests failed. Can not add token fee"
    exit 1
fi

##
## try execute tx with custom fee
##
ARGS_WITH_CUSTOM_FEE="--from $USER --home $NODE_HOME --keyring-backend test --chain-id $CHAIN_ID -y --gas auto --gas-adjustment 1.5 --fees 1000000$full_denom_name -b sync"
send_txhash=$(oraid tx bank send $user_address orai1kzkf6gttxqar9yrkxfe34ye4vg5v4m588ew7c9 10000$full_denom_name $ARGS_WITH_CUSTOM_FEE --output json| jq -r '.txhash')

sleep 2
code=$(oraid query tx $send_txhash --output json | jq -r '.code')
if ! [[ $code =~ 0 ]]; then
    echo "Txfees tests failed. Can not send tx with custom fee"
    exit 1
fi

echo "Txfees tests successfully!"
