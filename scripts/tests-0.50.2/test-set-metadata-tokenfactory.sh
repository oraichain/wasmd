#!/bin/bash

set -ux

CHAIN_ID=${CHAIN_ID:-testing}
USER=${USER:-"validator1"}
NODE_HOME=${NODE_HOME:-"$PWD/.oraid"}
ARGS="--from $USER --chain-id $CHAIN_ID -y --keyring-backend test --gas auto --gas-adjustment 1.5 -b sync --home $NODE_HOME"
HIDE_LOGS="/dev/null"

# prepare a new contract for gasless
fee_params=$(oraid query tokenfactory params --output json | jq '.params.denom_creation_fee[0].denom')
if ! [[ $fee_params =~ "orai" ]]; then
   echo "Tokenfactory set metadata tests failed. The tokenfactory fee params is not orai"
   exit 1
fi

# try creating a new denom
denom_name="usd"
oraid tx tokenfactory create-denom $denom_name $ARGS >$HIDE_LOGS

# try querying list denoms afterwards
# need to sleep 1s
sleep 1
user_address=$(oraid keys show $USER --home $NODE_HOME --keyring-backend test -a)
first_denom=$(oraid query tokenfactory denoms-from-creator $user_address --output json | jq '.denoms[0]' | tr -d '"')
echo "first denom: $first_denom"

if ! [[ $first_denom =~ "factory/$user_address/$denom_name" ]]; then
   echo "Tokenfactory set metadata tests failed. The tokenfactory denom does not match the created denom"
   exit 1
fi

admin=$(oraid query tokenfactory denom-authority-metadata $first_denom --output json | jq '.authority_metadata.admin')
echo "admin: $admin"

if ! [[ $admin =~ $user_address ]]; then
   echo "Tokenfactory set metadata tests failed. The tokenfactory admin does not match the creator"
   exit 1
fi

sleep 2
# try to set denom metadata
ticker="TICKER"
description="description"
exponent=6
oraid tx tokenfactory modify-metadata $first_denom $ticker $description $exponent $ARGS >$HIDE_LOGS

sleep 2
symbol=$(oraid query bank denom-metadata $first_denom --output json | jq '.metadata.symbol' |  tr -d '"')
if ! [[ $ticker =~ $symbol ]]; then
   echo "Tokenfactory set metadata tests failed. The tokenfactory ticker does not match symbol after modify metadata"
   exit 1
fi

echo "Tokenfactory set metadata tests passed!"
