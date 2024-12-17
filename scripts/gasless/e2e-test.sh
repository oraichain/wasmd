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
if ! [ -d "../orai-0424" ]; then
    git clone --branch v0.42.4 --single-branch https://github.com/oraichain/orai.git $PWD/../orai-0424
fi

# build old binary
CUR_DIR=$PWD && cd $PWD/../orai-0424 && go mod tidy && GOTOOLCHAIN=go1.21.4 make install && cd $CUR_DIR

# setup local network
sh $PWD/scripts/gasless/old-multinode-local-testnet.sh

# setup gasless
sleep 5
NODE_HOME=$VALIDATOR_HOME USER=validator1 WASM_PATH="$PWD/scripts/wasm_file/counter_high_gas_cost.wasm" source $PWD/scripts/gasless/create-gasless.sh

sleep 5
# create new upgrade proposal
latest_height=$(curl --no-progress-meter http://localhost:1316/cosmos/base/tendermint/v1beta1/blocks/latest | jq '.block.header.height | tonumber')
UPGRADE_HEIGHT=$(($latest_height + 30))
NEW_VERSION="v0.50.0"
oraid tx gov submit-proposal software-upgrade $NEW_VERSION --title "foobar" --description "foobar" --from validator1 --upgrade-height $UPGRADE_HEIGHT --upgrade-info "x" --deposit 10000000orai $ARGS --home $VALIDATOR_HOME >$HIDE_LOGS
sleep 1
oraid tx gov vote 1 yes --from validator1 --home $VALIDATOR_HOME $ARGS >$HIDE_LOGS && oraid tx gov vote 1 yes --from validator2 --home "$HOME/.oraid/validator2" $ARGS >$HIDE_LOGS

# sleep to wait til the proposal passes
echo "Sleep til the proposal passes..."

# Check if latest height is less than the upgrade height
latest_height=$(curl --no-progress-meter http://localhost:1317/cosmos/base/tendermint/v1beta1/blocks/latest | jq '.block.header.height | tonumber')
while [ $latest_height -lt $UPGRADE_HEIGHT ]; do
    sleep 5
    ((latest_height = $(curl --no-progress-meter http://localhost:1317/cosmos/base/tendermint/v1beta1/blocks/latest | jq '.block.header.height | tonumber')))
    echo $latest_height
done

# kill all processes
pkill oraid

# install new binary for the upgrade
# install new binary for the upgrade
echo "install new binary"
# clone or pull latest repo
if ! [ -d "$PWD/../orai-050" ]; then
    git clone https://github.com/oraichain/wasmd.git $PWD/../orai-050
fi
CUR_DIR=$PWD && cd $PWD/../orai-050 && git checkout $NEW_VERSION && go mod tidy && GOTOOLCHAIN=$GO_VERSION make build && cd $CUR_DIR

# re-run all validators. All should run
screen -S validator1 -d -m oraid start --home=$HOME/.oraid/validator1
screen -S validator2 -d -m oraid start --home=$HOME/.oraid/validator2
screen -S validator3 -d -m oraid start --home=$HOME/.oraid/validator3

# sleep a bit for the network to start
echo "Sleep to wait for the network to start..."
# sleep longer than usual for module migration
sleep 7

# sleep about 5 secs to wait for the rest & json rpc server to be u
echo "Waiting for the REST & JSONRPC servers to be up ..."
sleep 5

oraid_version=$(oraid version)
if [[ $oraid_version =~ $OLD_VERSION ]]; then
    echo "The chain has not upgraded yet. There's something wrong!"
    exit 1
fi

height_before=$(curl --no-progress-meter http://localhost:1317/cosmos/base/tendermint/v1beta1/blocks/latest | jq '.block.header.height | tonumber')

re='^[0-9]+([.][0-9]+)?$'
if ! [[ $height_before =~ $re ]]; then
    echo "error: Not a number" >&2
    exit 1
fi

sleep 5

height_after=$(curl --no-progress-meter http://localhost:1317/cosmos/base/tendermint/v1beta1/blocks/latest | jq '.block.header.height | tonumber')

if ! [[ $height_after =~ $re ]]; then
    echo "error: Not a number" >&2
    exit 1
fi

if [ $height_after -gt $height_before ]; then
    echo "Chain Upgrade Passed"
else
    echo "Chain Upgrade Failed"
fi

inflation=$(curl --no-progress-meter http://localhost:1317/cosmos/mint/v1beta1/inflation | jq '.inflation | tonumber')
if ! [[ $inflation =~ $re ]]; then
    echo "Error: Cannot query inflation => Potentially missing Go GRPC backport" >&2
    echo "Tests Failed"
    exit 1
fi

evm_denom=$(curl --no-progress-meter http://localhost:1317/ethermint/evm/v1/params | jq '.params.evm_denom')
if ! [[ $evm_denom =~ "aorai" ]]; then
    echo "Error: EVM denom is not correct. The upgraded version is not the latest!" >&2
    echo "Tests Failed"
    exit 1
fi

echo "gas used before: $gas_used_before"
sleep 5
contract_address=$contract_address gas_used_before=$gas_used_before code_id=$code_id USER=validator1 USER2=validator2 WASM_PATH="$PWD/scripts/wasm_file/counter_high_gas_cost.wasm" sh $PWD/scripts/gasless/test-gasless.sh

echo "Tests Passed!!"
bash scripts/clean-multinode-local-testnet.sh
