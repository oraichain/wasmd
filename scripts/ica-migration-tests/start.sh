#!/bin/bash

set -e

BINARY=oraid
CHAIN_DIR=$PWD/data
CHAINID_1=test-1
CHAINID_2=test-2


echo "Starting $CHAINID_1 in $CHAIN_DIR..."
echo "Creating log file at $CHAIN_DIR/$CHAINID_1.log"
$BINARY start --log_level info --log_format json --home $CHAIN_DIR/$CHAINID_1 --pruning=nothing > $CHAIN_DIR/$CHAINID_1.log 2>&1 &

echo "Starting $CHAINID_2 in $CHAIN_DIR..."
echo "Creating log file at $CHAIN_DIR/$CHAINID_2.log"
$BINARY start --log_level info --log_format json --home $CHAIN_DIR/$CHAINID_2 --pruning=nothing > $CHAIN_DIR/$CHAINID_2.log 2>&1 &