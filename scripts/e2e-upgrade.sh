#!/bin/bash

set -eu

# setup the network using the old binary

OLD_VERSION=${OLD_VERSION:-"v0.42.4"}
WASM_PATH=${WASM_PATH:-"$PWD/scripts/wasm_file/oraiswap-token.wasm"}
ARGS="--chain-id testing -y --keyring-backend test --gas auto --gas-adjustment 1.5"
NEW_VERSION=${NEW_VERSION:-"v0.50.0"}
VALIDATOR_HOME=${VALIDATOR_HOME:-"$HOME/.oraid/validator1"}
MIGRATE_MSG=${MIGRATE_MSG:-'{}'}
EXECUTE_MSG=${EXECUTE_MSG:-'{"ping":{}}'}
HIDE_LOGS="/dev/null"
GO_VERSION=$(go version | awk '{print $3}')

# kill all running binaries
pkill oraid && sleep 2

# download current production binary
current_dir=$PWD

# clone or pull latest repo
if ! [ -d "../orai-old" ]; then
  git clone https://github.com/oraichain/orai.git ../orai-old
fi

# build old binary
cd ../orai-old
git fetch
git checkout $OLD_VERSION
go mod tidy && GOTOOLCHAIN=$GO_VERSION make install

cd $current_dir

# setup local network
sh $PWD/scripts/multinode-local-testnet-v0.42.4.sh

# deploy new contract
store_ret=$(oraid tx wasm store $WASM_PATH --from validator1 --home $VALIDATOR_HOME $ARGS -b block --output json)
code_id=$(echo $store_ret | jq -r '.logs[0].events[1].attributes[] | select(.key | contains("code_id")).value')
INSTANTIATE_MSG='{"name":"OraichainToken","symbol":"ORAI","decimals":6,"initial_balances":[{"amount":"1000000000","address":"orai1kzkf6gttxqar9yrkxfe34ye4vg5v4m588ew7c9"},{"amount":"1000000000","address":"orai1hgscrqcd2kmju4t5akujeugwrfev7uxv66lnuu"}]}'
oraid tx wasm instantiate $code_id $INSTANTIATE_MSG --label 'testing' --from validator1 --home $VALIDATOR_HOME -b block --admin $(oraid keys show validator1 --keyring-backend test --home $VALIDATOR_HOME -a) $ARGS > $HIDE_LOGS
contract_address=$(oraid query wasm list-contract-by-code $code_id --output json | jq -r '.contracts[0]')

echo "contract address: $contract_address"

# create new upgrade proposal
UPGRADE_HEIGHT=${UPGRADE_HEIGHT:-35}
oraid tx gov submit-proposal software-upgrade $NEW_VERSION --title "foobar" --description "foobar" --from validator1 --upgrade-height $UPGRADE_HEIGHT --upgrade-info "x" --deposit 10000000orai $ARGS --home $VALIDATOR_HOME -b block > $HIDE_LOGS
oraid tx gov vote 1 yes --from validator1 --home $VALIDATOR_HOME $ARGS > $HIDE_LOGS && oraid tx gov vote 1 yes --from validator2 --home "$HOME/.oraid/validator2" $ARGS > $HIDE_LOGS

# sleep to wait til the proposal passes
echo "Sleep til the proposal passes..."

# Check if latest height is less than the upgrade height
latest_height=$(curl --no-progress-meter http://localhost:1317/blocks/latest | jq '.block.header.height | tonumber')
while [ $latest_height -lt $UPGRADE_HEIGHT ];
do
   sleep 5
   ((latest_height=$(curl --no-progress-meter http://localhost:1317/blocks/latest | jq '.block.header.height | tonumber')))
   echo $latest_height
done

# kill all processes
pkill oraid

# install new binary for the upgrade
echo "install new binary"
GOTOOLCHAIN=$GO_VERSION make build

# Back to current folder
cd $current_dir

# re-run all validators. All should run
screen -S validator1 -d -m oraid start --home=$HOME/.oraid/validator1
screen -S validator2 -d -m oraid start --home=$HOME/.oraid/validator2
screen -S validator3 -d -m oraid start --home=$HOME/.oraid/validator3

# sleep a bit for the network to start
echo "Sleep to wait for the network to start..."
# sleep longer than usual for module migration
sleep 7

# test contract migration
echo "Upload the new contract code id..."
upload_wasm_txhash=$(oraid tx wasm store $WASM_PATH --from validator1 --home $VALIDATOR_HOME $ARGS --output json | jq -r '.txhash')
sleep 2
new_code_id=$(oraid q wasm list-code --reverse --limit 1 --output json | jq -r '.code_infos[0].code_id')

echo "Migrate the contract..."
oraid tx wasm migrate $contract_address $new_code_id $MIGRATE_MSG --from validator1 $ARGS --home $VALIDATOR_HOME
sleep 2

# sleep about 5 secs to wait for the rest & json rpc server to be u
echo "Waiting for the REST & JSONRPC servers to be up ..."
sleep 5

oraid_version=$(oraid version)
if [[ $oraid_version =~ $OLD_VERSION ]] ; then
   echo "The chain has not upgraded yet. There's something wrong!"; exit 1
fi

height_before=$(curl --no-progress-meter http://localhost:1317/cosmos/base/tendermint/v1beta1/blocks/latest | jq '.block.header.height | tonumber')

re='^[0-9]+([.][0-9]+)?$'
if ! [[ $height_before =~ $re ]] ; then
   echo "error: Not a number" >&2; exit 1
fi

sleep 5

height_after=$(curl --no-progress-meter http://localhost:1317/cosmos/base/tendermint/v1beta1/blocks/latest | jq '.block.header.height | tonumber')

if ! [[ $height_after =~ $re ]] ; then
   echo "error: Not a number" >&2; exit 1
fi

if [ $height_after -gt $height_before ]
then
echo "Chain Upgrade Passed"
else
echo "Chain Upgrade Failed"
fi

inflation=$(curl --no-progress-meter http://localhost:1317/cosmos/mint/v1beta1/inflation | jq '.inflation | tonumber')
if ! [[ $inflation =~ $re ]] ; then
   echo "Error: Cannot query inflation => Potentially missing Go GRPC backport" >&2;
   echo "Tests Failed"; exit 1
fi

evm_denom=$(curl --no-progress-meter http://localhost:1317/ethermint/evm/v1/params | jq '.params.evm_denom')
if ! [[ $evm_denom =~ "aorai" ]] ; then
   echo "Error: EVM denom is not correct. The upgraded version is not the latest!" >&2;
   echo "Tests Failed"; exit 1
fi

sh $PWD/scripts/test_clock_counter_contract.sh

# test gasless
USER=validator1 USER2=validator2 WASM_PATH="$PWD/scripts/wasm_file/counter_high_gas_cost.wasm" sh $PWD/scripts/tests-0.42.1/test-gasless.sh
NODE_HOME=$VALIDATOR_HOME USER=validator1 sh $PWD/scripts/tests-0.42.1/test-tokenfactory.sh
NODE_HOME=$VALIDATOR_HOME USER=validator1 sh $PWD/scripts/tests-0.42.1/test-tokenfactory-bindings.sh
NODE_HOME=$VALIDATOR_HOME USER=validator1 sh $PWD/scripts/tests-0.42.1/test-evm-cosmos-mapping.sh
NODE_HOME=$VALIDATOR_HOME USER=validator1 sh $PWD/scripts/tests-0.42.1/test-evm-cosmos-mapping-complex.sh
NODE_HOME=$VALIDATOR_HOME USER=validator1 sh $PWD/scripts/tests-0.42.2/test-multi-sig.sh
NODE_HOME=$VALIDATOR_HOME sh $PWD/scripts/tests-0.42.3/test-commit-timeout.sh
NODE_HOME=$VALIDATOR_HOME sh $PWD/scripts/tests-0.42.4/test-cw-stargate-staking-query.sh
NODE_HOME=$VALIDATOR_HOME USER=validator1 sh $PWD/scripts/tests-0.42.4/test-cw20-erc20.sh
NODE_HOME=$VALIDATOR_HOME USER=validator1 sh $PWD/scripts/tests-0.42.4/test-globalfee.sh

bash scripts/clean-multinode-local-testnet.sh
echo "Tests Passed!!"
