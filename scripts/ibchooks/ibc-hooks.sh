#!/bin/bash
#
# This script can be used to manually test the ibc hooks. It is meant as a guide and not to be run directly
# without taking into account the context in which it's being run.
# The script uses `jenv` (https://github.com/nicolaslara/jenv) to easily generate the json strings passed
# to some of the commands. If you don't want to use it you can generate the json manually or modify this script.
#
set -o errexit -o nounset -o pipefail -o xtrace
shopt -s expand_aliases

alias chainA="oraid --node http://localhost:26657 --chain-id localoraichain-a"
alias chainB="oraid --node http://localhost:36657 --chain-id localoraichain-b"
alias chainAWithoutChainId="oraid --node http://localhost:26657"
alias chainBWithoutChainId="oraid --node http://localhost:36657"

# setup the keys
echo "bottom loan skill merry east cradle onion journey palm apology verb edit desert impose absurd oil bubble sweet glove shallow size build burst effort" | oraid keys add validator --keyring-backend test --recover || echo "key exists"
echo "increase bread alpha rigid glide amused approve oblige print asset idea enact lawn proof unfold jeans rabbit audit return chuckle valve rather cactus great" | oraid keys add faucet --keyring-backend test --recover || echo "key exists"

VALIDATOR=$(oraid keys show validator -a --keyring-backend test)

args="--keyring-backend test --log_level=debug --gas auto --gas-prices 0.1orai --gas-adjustment 1.3 --broadcast-mode sync --yes"
TX_FLAGS=($args)

# send money to the validator on both chains
chainA tx bank send faucet "$VALIDATOR" 1000000000orai "${TX_FLAGS[@]}"
sleep 3
chainB tx bank send faucet "$VALIDATOR" 1000000000orai "${TX_FLAGS[@]}"
sleep 3

# store and instantiate the contract
chainA tx wasm store ./bytecode/counter.wasm --from validator "${TX_FLAGS[@]}"
sleep 3
CONTRACT_ID=$(chainAWithoutChainId query wasm list-code -o json | jq -r '.code_infos[-1].code_id')
chainA tx wasm instantiate "$CONTRACT_ID" '{"count": 0}' --from validator --no-admin --label=counter "${TX_FLAGS[@]}"
sleep 3

# get the contract address
export CONTRACT_ADDRESS=$(chainAWithoutChainId query wasm list-contract-by-code 1 -o json | jq -r '.contracts | [last][0]')

origin_denom=$(chainAWithoutChainId query bank balances "$CONTRACT_ADDRESS" -o json | jq -r '.balances[0].denom')
balance=$(chainAWithoutChainId query bank balances "$CONTRACT_ADDRESS" -o json | jq -r '.balances[0].amount')

QUERY="{\"get_count\": { }}"
count_before=$(chainAWithoutChainId query wasm contract-state smart "$CONTRACT_ADDRESS" "$QUERY" -o json | jq -r '.data.count')

# send ibc transaction to execite the contract
MEMO='{"wasm":{"contract":"'"$CONTRACT_ADDRESS"'","msg": {"increment": {}} }}'
chainB tx ibc-transfer transfer transfer channel-0 $CONTRACT_ADDRESS 10orai \
       --from validator \
       "${TX_FLAGS[@]}" \
       --memo "$MEMO"

# wait for the ibc round trip
sleep 16

new_balance=$(chainAWithoutChainId query bank balances "$CONTRACT_ADDRESS" -o json | jq -r '.balances[0].amount')
denom=$(chainAWithoutChainId query bank balances "$CONTRACT_ADDRESS" -o json | jq -r '.balances[0].denom')

export ADDR_IN_CHAIN_A=$(chainAWithoutChainId q ibchooks wasm-sender channel-0 "$VALIDATOR")
# QUERY="{\"get_total_funds\": {\"addr\": \"$ADDR_IN_CHAIN_A\"}}"
# funds=$(chainAWithoutChainId query wasm contract-state smart "$CONTRACT_ADDRESS" "$QUERY" -o json | jq -c -r '.data.total_funds[]')
count=$(chainAWithoutChainId query wasm contract-state smart "$CONTRACT_ADDRESS" "$QUERY" -o json | jq -r '.data.count')

echo "count before: $count_before"
echo "count after: $count"
echo "origin_denom: $origin_denom, old balance: $balance, denom: $denom, new balance: $new_balance"
