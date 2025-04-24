#!/bin/bash
# Before running this script, you must setup local network:
# sh $PWD/scripts/multinode-local-testnet.sh
# oraiswap-token.wasm source code: https://github.com/oraichain/oraiswap.git
# tokenfactory.wasm source code: https://github.com/oraichain/token-bindings.git

set -ux

# hard-coded test private key. DO NOT USE!!
PRIVATE_KEY_ETH=${PRIVATE_KEY_ETH:-"021646C7F742C743E60CC460C56242738A3951667E71C803929CB84B6FA4B0D6"}
PRIVATE_KEY_EVM_ADDRESS=${PRIVATE_KEY_EVM_ADDRESS:-"0xB0ac9d216b303a32907632731a93356228CAEE87"}
current_dir=$PWD
WASM_PATH=${WASM_PATH:-"$PWD/scripts/wasm_file/oraiswap-token.wasm"}
TOKEN_FACTORY_WASM_PATH=${TOKEN_FACTORY_WASM_PATH:-"$PWD/scripts/wasm_file/tokenfactory.wasm"}
NODE_HOME=${NODE_HOME:-"$PWD/.oraid"}
USER=${USER:-"validator1"}
ARGS="--from $USER --home $NODE_HOME --chain-id testing -y --keyring-backend test --gas auto --gas-adjustment 1.5 --fees 1000orai -b sync"
HIDE_LOGS="/dev/null"

store_ret=$(oraid tx wasm store $WASM_PATH $ARGS --output json)
store_txhash=$(echo $store_ret | jq -r '.txhash')
# need to sleep 1s for tx already in block
sleep 2
code_id=$(oraid query tx $store_txhash --output json | jq -r '.events[] | select(.type == "store_code") | .attributes[] | select(.key == "code_id") | .value')

# 2 addresses that map with the hard-coded private key above
INSTANTIATE_MSG='{"name":"OraichainToken","symbol":"ORAI","decimals":6,"initial_balances":[{"amount":"1000000000","address":"orai1kzkf6gttxqar9yrkxfe34ye4vg5v4m588ew7c9"},{"amount":"1000000000","address":"orai1hgscrqcd2kmju4t5akujeugwrfev7uxv66lnuu"}]}'
oraid tx wasm instantiate $code_id $INSTANTIATE_MSG --label 'cw20 ORAI' --admin $(oraid keys show $USER --keyring-backend test --home $NODE_HOME -a) $ARGS > $HIDE_LOGS
# need to sleep 1s for tx already in block
sleep 2
contract_address=$(oraid query wasm list-contract-by-code $code_id --output json | jq -r '.contracts | last')
echo "cw-stargate-staking-query contract address: $contract_address"

# clone or pull latest repo
if [ -d "$PWD/../evm-entry-point" ]; then
  cd ../evm-entry-point
  git checkout chore/test-evm-precompile
  git pull origin chore/test-evm-precompile
else
  git clone https://github.com/oraidex/evm-entry-point.git ../evm-entry-point
  cd ../evm-entry-point
  git checkout chore/test-evm-precompile
  git pull origin chore/test-evm-precompile
fi

# prepare env and chain
pnpm install && cd packages/contracts && pnpm compile;
echo "PRIVATE_KEY=$PRIVATE_KEY_ETH" > .env

# before deploying erc20, we need to fund the private key's address first
oraid tx bank send $USER orai1kzkf6gttxqar9yrkxfe34ye4vg5v4m588ew7c9 100000orai $ARGS > $HIDE_LOGS
sleep 2 # wait for tx

# deploy cw20erc20 contract
echo "Deploying cw20erc20 contract..."
output=$(CW20_ADDRESS=$contract_address pnpm hardhat run scripts/create-cw20-erc20.ts --network testing)
echo "Deploying cw20erc20 contract: Passed"


# deploy cw-bindings contract
store_txhash=$(oraid tx wasm store $TOKEN_FACTORY_WASM_PATH $ARGS --output json | jq -r '.txhash')
# need to sleep 1s
sleep 2
code_id=$(oraid query tx $store_txhash --output json | jq -r '.events[] | select(.type == "store_code") | .attributes[] | select(.key == "code_id") | .value')
user_address=$(oraid keys show $USER --keyring-backend test --home $NODE_HOME -a)
oraid tx wasm instantiate $code_id '{}' --label 'tokenfactory cw bindings testing' --admin $user_address $ARGS >$HIDE_LOGS
sleep 2
token_factory_contract_address=$(oraid query wasm list-contract-by-code $code_id --output json | jq -r '.contracts[0]')
echo "token factory contract address: $token_factory_contract_address"
sleep 2


# create erc20 native and transfer from token
echo "Creating erc20 native and transferring from token..."
output=$(TOKEN_FACTORY_ADDRESS=$token_factory_contract_address pnpm hardhat run scripts/create-erc20-native-and-transfer-from-token.ts --network testing)
sleep 2

# create erc20 native and transfer
echo "Creating erc20 native and transferring token..."
output=$(TOKEN_FACTORY_ADDRESS=$token_factory_contract_address pnpm hardhat run scripts/create-erc20-native-and-transfer-token.ts --network testing)
sleep 2

# create native erc20 upgradeable
echo "Creating native erc20 upgradeable..."
output=$(TOKEN_FACTORY_ADDRESS=$token_factory_contract_address pnpm hardhat run scripts/create-native-erc20-upgradeable.ts --network testing)
sleep 2

# create native erc20 
echo "Creating native erc20..."
output=$(TOKEN_FACTORY_ADDRESS=$token_factory_contract_address pnpm hardhat run scripts/create-native-erc20.ts --network testing)

echo "Test EVM entry point: Passed!"
