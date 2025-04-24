#!/bin/bash

set -ux

# ------------------------------------------------------------------------------------------------
# Setup chain with old binary
# ------------------------------------------------------------------------------------------------

# setup the network using the old binary
OLD_VERSION=${OLD_VERSION:-"v0.50.10"}
ARGS="--chain-id testing -y --keyring-backend test --gas auto --gas-adjustment 1.5"
NEW_VERSION=${NEW_VERSION:-"v0.50.11"}
HIDE_LOGS="/dev/null"
GO_VERSION=$(go version | awk '{print $3}')

# kill all running binaries
pkill oraid && sleep 2

# download current production binary
current_dir=$PWD

# clone or pull latest repo
if ! [ -d "../orai-old" ]; then
   git clone https://github.com/oraichain/wasmd.git ../orai-old
fi

# build old binary
cd ../orai-old
git fetch
git checkout $OLD_VERSION
go mod tidy && GOTOOLCHAIN=$GO_VERSION make build

cd $current_dir

# setup local network
sh $PWD/scripts/multinode-local-testnet.sh

sleep 5

# ------------------------------------------------------------------------------------------------
# Setup evm contract
# ------------------------------------------------------------------------------------------------

# hard-coded test private key. DO NOT USE!!
PRIVATE_KEY_ETH=${PRIVATE_KEY_ETH:-"021646C7F742C743E60CC460C56242738A3951667E71C803929CB84B6FA4B0D6"}
PRIVATE_KEY_EVM_ADDRESS=${PRIVATE_KEY_EVM_ADDRESS:-"0xB0ac9d216b303a32907632731a93356228CAEE87"}
VALIDATOR1_ARGS=${VALIDATOR1_ARGS:-"--from validator1 --home $HOME/.oraid/validator1"}
USER="validator1"

# Use local evm-contracts directory
cd $PWD/scripts/evm-contracts/counter

# prepare env and chain
yarn && yarn compile;
echo "PRIVATE_KEY=$PRIVATE_KEY_ETH" > .env

# before deploying counter contract, we need to fund the private key's address first
oraid tx bank send $USER orai1kzkf6gttxqar9yrkxfe34ye4vg5v4m588ew7c9 100000orai $VALIDATOR1_ARGS $ARGS > $HIDE_LOGS
sleep 2 # wait for tx

# deploy counter contract
output=$(yarn hardhat run scripts/deploy-counter.ts --network testing)
# collect only the contract address part
contract_addr=$(echo "$output" | grep -oE '0x[0-9a-fA-F]+')
echo "Counter contract addr: $contract_addr"

# validate
contract_addr_len=${#contract_addr}
if [ $contract_addr_len -ne 42 ] ; then
   echo "Couldn't deploy counter contract. Upstream EVM Test Failed";
   exit 1
fi

# try querying counter value
output=$(COUNTER_ADDRESS=$contract_addr yarn hardhat run scripts/query-counter.ts --network testing)
counter_value=$(echo "$output" | awk '/^[0-9]+$/ { print $1 }')
if ! [ $counter_value == "10" ] ; then
   echo "Could not query counter value. Upstream EVM Test Failed"; 
   exit 1
fi

# try to increment the counter
output=$(COUNTER_ADDRESS=$contract_addr yarn hardhat run scripts/increase-counter.ts --network testing)
echo "Increment counter output: $output"

cd $current_dir

# ------------------------------------------------------------------------------------------------
# Upgrade chain
# ------------------------------------------------------------------------------------------------

# create new upgrade proposal
UPGRADE_HEIGHT=${UPGRADE_HEIGHT:-80}

VERSION=$NEW_VERSION HEIGHT=$UPGRADE_HEIGHT bash $PWD/scripts/proposal-script.sh

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
echo "install new binary"
GOTOOLCHAIN=$GO_VERSION make build

sleep 5
# Back to current folder
cd $current_dir

# re-run all validators. All should run
screen -S validator1 -d -m oraid start --home=$HOME/.oraid/validator1
screen -S validator2 -d -m oraid start --home=$HOME/.oraid/validator2
screen -S validator2 -d -m oraid start --home=$HOME/.oraid/validator3

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

evm_denom=$(curl --no-progress-meter http://localhost:1317/cosmos/evm/vm/v1/params | jq '.params.evm_denom')
if ! [[ $evm_denom =~ "aorai" ]]; then
   echo "Error: EVM denom is not correct. The upgraded version is not the latest!" >&2
   echo "Tests Failed"
   exit 1
fi

# ------------------------------------------------------------------------------------------------
# Test counter contract
# ------------------------------------------------------------------------------------------------

cd $PWD/scripts/evm-contracts/counter
# try querying counter value
output=$(COUNTER_ADDRESS=$contract_addr yarn hardhat run scripts/query-counter.ts --network testing)
counter_value=$(echo "$output" | awk '/^[0-9]+$/ { print $1 }')
if ! [ $counter_value == "11" ] ; then
   echo "Could not query counter value. Upstream EVM Test Failed"; 
   exit 1
fi

echo "Upstream EVM Test Passed"; 
bash scripts/clean-multinode-local-testnet.sh







