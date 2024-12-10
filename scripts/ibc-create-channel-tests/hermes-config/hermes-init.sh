#!/bin/bash
set -ux

# Configure predefined mnemonic pharses
BINARY=hermes
CHAIN_DIR=$PWD/data
CHAINID_1=test-1
CHAINID_2=test-2
CLIENT_ID="07-tendermint-0"
RELAYER_DIR=hermes
MNEMONIC_1="alley afraid soup fall idea toss can goose become valve initial strong forward bright dish figure check leopard decide warfare hub unusual join cart"
MNEMONIC_2="record gift you once hip style during joke field prize dust unique length more pencil transfer quit train device arrive energy sort steak upset"

export PATH="$HOME/.hermes/bin:$PATH"
# Ensure hermes is installed
if ! [ -x "$(command -v $BINARY)" ]; then
    echo "$BINARY is required to run this script..."
    exit 1
fi

# add key for chain 1
echo "Adding key for $CHAINID_1"
mkdir -p $CHAIN_DIR/$RELAYER_DIR/mnemonics/
echo $MNEMONIC_1 >$CHAIN_DIR/$RELAYER_DIR/mnemonics/$CHAINID_1
$BINARY keys add \
    --chain $CHAINID_1 \
    --mnemonic-file $CHAIN_DIR/$RELAYER_DIR/mnemonics/$CHAINID_1 \
    --key-name $CHAINID_1 \
    --overwrite

# add key for chain 2
echo "Adding key for $CHAINID_2"
echo $MNEMONIC_2 >$CHAIN_DIR/$RELAYER_DIR/mnemonics/$CHAINID_2
$BINARY keys add \
    --chain $CHAINID_2 \
    --mnemonic-file $CHAIN_DIR/$RELAYER_DIR/mnemonics/$CHAINID_2 \
    --key-name $CHAINID_2 \
    --overwrite

# wait 2 chains to already start
sleep 10

# create new hermes client
$BINARY create client \
    --host-chain $CHAINID_1 \
    --reference-chain $CHAINID_2

$BINARY create client \
    --host-chain $CHAINID_2 \
    --reference-chain $CHAINID_1

# create new hermes connection
$BINARY create connection \
    --a-chain $CHAINID_1 \
    --a-client $CLIENT_ID \
    --b-client $CLIENT_ID
