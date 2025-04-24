#!/bin/bash

set -eu

# setup the network using the old binary

BASEDIR=$(dirname $0)
PROJECT_DIR=$(realpath "$BASEDIR/..")
cd $PROJECT_DIR

WASM_PATH=${WASM_PATH:-"scripts/wasm_file/swapmap.wasm"}
ARGS="--chain-id testing -y --keyring-backend test --gas auto --gas-adjustment 1.5"
VALIDATOR_HOME=${VALIDATOR_HOME:-"$HOME/.oraid/validator1"}
re='^[0-9]+([.][0-9]+)?$'

# rebuild the latest code before testing
make build

# setup local network
bash scripts/multinode-local-testnet.sh

# sleep about 5 secs to wait for the rest & json rpc server to be u
echo "Waiting for the REST & JSONRPC servers to be up ..."
{
  while ! echo -n > /dev/tcp/localhost/1317; do
    sleep 1
  done
} 2>/dev/null

sleep 1
inflation=$(curl --no-progress-meter http://localhost:1317/cosmos/mint/v1beta1/inflation | jq '.inflation | tonumber')
if ! [[ $inflation =~ $re ]] ; then
   echo "Error: Cannot query inflation => Potentially missing Go GRPC backport" >&2;
   echo "Tests Failed"; exit 1
fi

evm_denom=$(curl --no-progress-meter http://localhost:1317/cosmos/evm/vm/v1/params | jq '.params.evm_denom')
if ! [[ $evm_denom =~ "aorai" ]] ; then
   echo "Error: EVM denom is not correct. The current chain version is not the latest!" >&2;
   echo "Tests Failed"; exit 1
fi

bash scripts/test_clock_counter_contract.sh

# v0.42.1 tests
# USER=validator1 USER2=validator2 WASM_PATH="$PROJECT_DIR/scripts/wasm_file/counter_high_gas_cost.wasm" bash scripts/tests-0.42.1/test-gasless.sh
NODE_HOME=$VALIDATOR_HOME USER=validator1 bash scripts/tests-0.42.1/test-tokenfactory.sh
NODE_HOME=$VALIDATOR_HOME USER=validator1 bash scripts/tests-0.42.1/test-tokenfactory-bindings.sh
NODE_HOME=$VALIDATOR_HOME USER=validator1 bash scripts/tests-0.42.1/test-evm-cosmos-mapping.sh
NODE_HOME=$VALIDATOR_HOME USER=validator1 bash scripts/tests-0.42.1/test-evm-cosmos-mapping-complex.sh

# v0.42.2 tests 
NODE_HOME=$VALIDATOR_HOME USER=validator1 bash scripts/tests-0.42.2/test-multi-sig.sh

# v0.42.3 tests
NODE_HOME=$VALIDATOR_HOME bash scripts/tests-0.42.3/test-commit-timeout.sh

# v0.42.4 tests
NODE_HOME=$VALIDATOR_HOME bash scripts/tests-0.42.4/test-cw-stargate-staking-query.sh
# NODE_HOME=$VALIDATOR_HOME USER=validator1 bash scripts/tests-0.42.4/test-cw20-erc20.sh
NODE_HOME=$VALIDATOR_HOME USER=validator1 bash scripts/tests-0.42.4/test-globalfee.sh

# v0.50.1 tests
bash $PWD/scripts/tests-0.50.1/test-mint-params.sh
bash $PWD/scripts/tests-0.50.1/test-gov-params.sh

# v0.50.2 tests
NODE_HOME=$VALIDATOR_HOME USER=validator1 sh $PWD/scripts/tests-0.50.2/test-set-metadata-tokenfactory.sh
NODE_HOME=$VALIDATOR_HOME USER=validator1 sh $PWD/scripts/tests-0.50.2/test-param-change-proposal-tokenfactory.sh

# v0.50.3 tests
NODE_HOME=$VALIDATOR_HOME USER=validator1 FUND=1000orai sh $PWD/scripts/tests-0.50.3/test-tokenfactory-metadata-binding.sh
USER=validator1 USER2=validator2 sh $PWD/scripts/tests-0.50.3/test-gasless.sh

# v0.50.4 tests
NODE_HOME=$VALIDATOR_HOME USER=validator1 FUND=1000orai sh $PWD/scripts/tests-0.50.4/test-tokenfactory-force-transfer.sh

# v0.50.9 tests
NODE_HOME=$VALIDATOR_HOME USER=validator1 sh $PWD/scripts/tests-0.50.9/test-txfees.sh
NODE_HOME=$VALIDATOR_HOME USER=validator1 sh $PWD/scripts/tests-0.50.9/test-evm-cosmos-mapping.sh
sh $PWD/scripts/tests-0.50.9/test-payable-with-bank-send.sh

echo "Tests Passed!!"
bash scripts/clean-multinode-local-testnet.sh
