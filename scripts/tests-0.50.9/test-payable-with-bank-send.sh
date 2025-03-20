#!/bin/bash
# Before running this script, you must setup local network:
# sh $PWD/scripts/multinode-local-testnet.sh
# oraiswap-token.wasm source code: https://github.com/oraichain/oraiswap.git

set -ux

# hard-coded test private key. DO NOT USE!!
current_dir=$PWD
ARGS="--chain-id testing -y --keyring-backend test --gas auto --gas-adjustment 1.5 --fees 10000orai -b sync"
VALIDATOR1_ARGS=${VALIDATOR1_ARGS:-"--from validator1 --home $HOME/.oraid/validator1"}

HIDE_LOGS="/dev/null"

# clone or pull latest repo
if [ -d "$PWD/../evm-entry-point" ]; then
  cd ../evm-entry-point
  # FIXME: change this to master once the PR is merged
  git checkout chore/test-payable
else
  git clone https://github.com/oraidex/evm-entry-point.git ../evm-entry-point
  cd ../evm-entry-point
  # FIXME: change this to master once the PR is merged
  git checkout chore/test-payable
fi

# prepare env and chain
npm install -g pnpm
pnpm install
cp packages/contracts/.env.example packages/contracts/.env

# compile the contracts for testing
cd packages/contracts && pnpm compile

# run the tests
result=$(pnpm hardhat run scripts/deploy-and-send-native.ts --network testing)

# check if the test passed
if echo "$result" | grep -q "Error deploying and sending native tokens:"; then
  echo "Test FAILED: Found error message in the output"
  echo "$result"
  cd $current_dir
  exit 1
else
  echo "Test payable with bank send PASSED"
  cd $current_dir
  exit 0
fi