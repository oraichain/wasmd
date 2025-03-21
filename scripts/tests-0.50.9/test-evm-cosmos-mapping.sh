#!/bin/bash

set -eu

CHAIN_ID=${CHAIN_ID:-testing}
USER=${USER:-tupt}
NODE_HOME=${NODE_HOME:-"$PWD/.oraid"}
ARGS="--from $USER --chain-id $CHAIN_ID -y --keyring-backend test --gas auto --gas-adjustment 1.5 --fees 10000orai -b sync --home $NODE_HOME"
HIDE_LOGS="/dev/null"

user_address="orai1knzg7jdc49ghnc2pkqg6vks8ccsk6efzfgv6gv"
before_address="orai18n4dmxpjrwx0dlst6h7zu9r628guwrrgjxchdl"
user_pubkey="AvSl0d9JrHCW4mdEyHvZu076WxLgH0bBVLigUcFm4UjV"
expected_evm_address="0x3ceADD98321b8CF6FE0BD5fC2E147A51d1c70c68"

# send coin to before_address
oraid tx bank send $USER $before_address 1orai $ARGS >$HIDE_LOGS
sleep 2

# query balance of before_address and user_address before mapping
before_balance=$(oraid query bank balance $before_address orai --output json | jq '.balance.amount | tonumber')
if [[ $before_balance -ne 1 ]]; then
    echo "EVM cosmos mapping test failed. The before_address balance does not increase after send coin"
    exit 1
fi

user_balance=$(oraid query bank balance $user_address orai --output json | jq '.balance.amount | tonumber')
if [[ $user_balance -ne 0 ]]; then
    echo "EVM cosmos mapping test failed. The user_address balance does not equal 0 before mapping"
    exit 1
fi

# mapping evm address to cosmos address
oraid tx evm set-mapping-evm $user_pubkey $ARGS &>$HIDE_LOGS
# wait for the tx to be completed
sleep 2

actual_evm_address=$(oraid query evm mappedevm $user_address --output json | jq '.evm_address' | tr -d '"')
if ! [[ $actual_evm_address =~ $expected_evm_address ]]; then
   echo "EVM cosmos mapping test failed. The evm addresses dont match"
   exit 1
fi

# wait for the jsonrpc to start
sleep 14

# query balance of before_address and user_address before mapping
before_balance=$(oraid query bank balance $before_address orai --output json | jq '.balance.amount | tonumber')
if [[ $before_balance -ne 0 ]]; then
    echo "EVM cosmos mapping test failed. The before_address balance does not equal 0 after mapping"
    exit 1
fi

user_balance=$(oraid query bank balance $user_address orai --output json | jq '.balance.amount | tonumber')
if [[ $user_balance -ne 1 ]]; then
    echo "EVM cosmos mapping test failed. The user_address balance does not equal 1 after mapping"
    exit 1
fi

# compare evm balance and cosmos balance
balance_hex=$(curl --no-progress-meter http://localhost:8545/ -X POST -H "Content-Type: application/json" --data '{"method":"eth_getBalance","params":["'"$actual_evm_address"'", "latest"],"id":1,"jsonrpc":"2.0"}' | jq '.result' | bc)
balance_hex_no_prefix=${balance_hex#0x}
balance_hex_no_prefix_upper=$(echo "$balance_hex_no_prefix" | tr '[:lower:]' '[:upper:]')
balance_decimal=$(echo "ibase=16; ${balance_hex_no_prefix_upper}" | bc)
echo "balance: $balance_decimal"

evm_decimals="10^18"
cosmos_decimals="10^6"
evm_balance=$(echo "scale=10; ($balance_decimal / $evm_decimals) * $cosmos_decimals" | bc)
evm_balance_int=$(echo ${evm_balance%.*})
if [[ $evm_balance_int -ne $user_balance ]]; then
   echo "EVM cosmos mapping test failed.The evm addresses dont match"
   exit 1
fi

echo "EVM cosmos mapping tests passed!"
