#!/bin/bash

BINARY=${BINARY:-oraid}
CHAIN_DIR=$PWD/data
CHAINID=${CHAINID:-"test-1"}


echo "Starting $CHAINID in $CHAIN_DIR..."
echo "Creating log file at $CHAIN_DIR/$CHAINID.log"
$BINARY start --log_level info --log_format json --home $CHAIN_DIR/$CHAINID --pruning=nothing > $CHAIN_DIR/$CHAINID_1.log 2>&1 &