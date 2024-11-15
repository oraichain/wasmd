#!/bin/bash

# Configure predefined mnemonic pharses
BINARY=rly
CHAIN_DIR=$PWD/data
CHAINID_1=test-1
CHAINID_2=test-2
RELAYER_DIR=./relayer
MNEMONIC_1="alley afraid soup fall idea toss can goose become valve initial strong forward bright dish figure check leopard decide warfare hub unusual join cart"
MNEMONIC_2="record gift you once hip style during joke field prize dust unique length more pencil transfer quit train device arrive energy sort steak upset"

# Ensure rly is installed
if ! [ -x "$(command -v $BINARY)" ]; then
    echo "$BINARY is required to run this script..."
    echo "You can download at https://github.com/cosmos/relayer"
    exit 1
fi

echo "Initializing $BINARY..."
$BINARY config init --home $CHAIN_DIR/$RELAYER_DIR

echo "Adding configurations for both chains..."
$BINARY chains add-dir $PWD/scripts/ica-migration-tests/go-rly-config/chains --home $CHAIN_DIR/$RELAYER_DIR
$BINARY paths add $CHAINID_1 $CHAINID_2 test1-test2 --file $PWD/scripts/ica-migration-tests/go-rly-config/paths/test1-test2.json --home $CHAIN_DIR/$RELAYER_DIR

echo "Restoring accounts..."
$BINARY keys restore $CHAINID_1 testkey "$MNEMONIC_1" --home $CHAIN_DIR/$RELAYER_DIR
$BINARY keys restore $CHAINID_2 testkey "$MNEMONIC_2" --home $CHAIN_DIR/$RELAYER_DIR

echo "Creating clients and a connection..."
$BINARY tx connection test1-test2 --home $CHAIN_DIR/$RELAYER_DIR