#!/bin/bash
set -ux

BINARY=oraid
OLD_BINARY=oraid-0424

CONNECTION_ID="connection-0"
CHAIN_ID1="test-1"
CHAIN_ID2="test-2"
CHANNEL1_ID="channel-1"
CHANNEL2_ID="channel-0"

CONFIG_DIR="$PWD/scripts/ibc-create-channel-tests"
DATA_DIR="$PWD/data"
GO_VERSION=$(go version | awk '{print $3}')
GOPATH="$HOME/go"

rm -rf $DATA_DIR
pkill oraid
pkill hermes

# clone or pull latest repo
if ! [ -d "$PWD/../orai-0424" ]; then
    git clone --branch v0.42.4 --single-branch https://github.com/oraichain/orai.git $PWD/../orai-0424
fi
CUR_DIR=$PWD && cd $PWD/../orai-0424 && go mod tidy && GOTOOLCHAIN=go1.21.4 make install && cp $GOPATH/bin/oraid $GOPATH/bin/$OLD_BINARY && cd $CUR_DIR

echo "Initializing both blockchains..."
bash $CONFIG_DIR/chains/init.sh
sh $CONFIG_DIR/chains/start.sh

# wait for nodes to start
echo "Initializing relayer..."
bash $CONFIG_DIR/hermes-config/hermes-init.sh

# start relayer
bash $CONFIG_DIR/hermes-config/hermes-start.sh >$DATA_DIR/hermes/hermes.log 2>&1 &

# wait for hermes rly to be up
sleep 5

# Store the following account addresses within the current shell env
export WALLET_1=$($BINARY keys show wallet1 -a --keyring-backend test --home $DATA_DIR/test-1) && echo $WALLET_1
export WALLET_2=$($BINARY keys show wallet2 -a --keyring-backend test --home $DATA_DIR/test-1) && echo $WALLET_2
export VAL_1=$($BINARY keys show val1 -a --keyring-backend test --home $DATA_DIR/test-1) && echo $VAL_1
export WALLET_3=$($BINARY keys show wallet3 -a --keyring-backend test --home $DATA_DIR/test-2) && echo $WALLET_3
export WALLET_4=$($BINARY keys show wallet4 -a --keyring-backend test --home $DATA_DIR/test-2) && echo $WALLET_4

# register ICA account
# Register an interchain account on behalf of WALLET_1 where chain test-2 is the interchain accounts host
$BINARY tx intertx register --from $WALLET_1 --connection-id connection-0 --chain-id test-1 --home $DATA_DIR/test-1 --node tcp://localhost:16657 --keyring-backend test -y -b block
$BINARY tx intertx register --from $WALLET_2 --connection-id connection-0 --chain-id test-1 --home $DATA_DIR/test-1 --node tcp://localhost:16657 --keyring-backend test -y -b block

# wait for hermes rly to craete new interchain acc
sleep 5

CHAIN1_PORT="icacontroller-$WALLET_1"
CHAIN2_PORT="icahost"
# create hermes channel
hermes create channel \
    --order ordered \
    --a-chain $CHAIN_ID1 \
    --a-connection $CONNECTION_ID \
    --a-port $CHAIN1_PORT \
    --b-port $CHAIN2_PORT

################################ upgrade chain
latest_height=$(curl --no-progress-meter http://localhost:1316/cosmos/base/tendermint/v1beta1/blocks/latest | jq '.block.header.height | tonumber')
UPGRADE_HEIGHT=$(($latest_height + 30))
NEW_VERSION="v0.50.0"
$BINARY tx gov submit-proposal software-upgrade $NEW_VERSION --title "foobar" --description "foobar" --from $WALLET_1 --upgrade-height $UPGRADE_HEIGHT --upgrade-info "x" --deposit 10000000orai --chain-id test-1 --home $DATA_DIR/test-1 --node tcp://localhost:16657 --keyring-backend test -y -b block
$BINARY tx gov vote 1 yes --from $VAL_1 --chain-id test-1 --home $DATA_DIR/test-1 --node tcp://localhost:16657 --keyring-backend test -y -b block

# sleep to wait til the proposal passes
echo "Sleep til the proposal passes..."

# Check if latest height is less than the upgrade height
latest_height=$(curl --no-progress-meter http://localhost:1316/cosmos/base/tendermint/v1beta1/blocks/latest | jq '.block.header.height | tonumber')
while [ $latest_height -lt $UPGRADE_HEIGHT ]; do
    sleep 5
    ((latest_height = $(curl --no-progress-meter http://localhost:1316/cosmos/base/tendermint/v1beta1/blocks/latest | jq '.block.header.height | tonumber')))
    echo $latest_height
done

# kill all processes
pkill $BINARY

# install new binary for the upgrade
echo "install new binary"
# clone or pull latest repo
if ! [ -d "$PWD/../orai-050" ]; then
    git clone https://github.com/oraichain/wasmd.git $PWD/../orai-050
fi
CUR_DIR=$PWD && cd $PWD/../orai-050 && git checkout $NEW_VERSION && go mod tidy && GOTOOLCHAIN=$GO_VERSION make build && cd $CUR_DIR

# re-run the nodes
bash $CONFIG_DIR/chains/start-single.sh
# start node 2 using different binary
CHAINID=test-2 BINARY=$OLD_BINARY bash $CONFIG_DIR/chains/start-single.sh

sleep 5

update_proposal() {
    cat $CONFIG_DIR/proposals/proposal.json | jq "$1" >$CONFIG_DIR/proposals/temp_proposal.json && mv $CONFIG_DIR/proposals/temp_proposal.json $CONFIG_DIR/proposals/proposal.json
}

################################ upgrade again to v0501
NEW_VERSION="v0.50.1"
latest_height=$(curl --no-progress-meter http://localhost:1316/cosmos/base/tendermint/v1beta1/blocks/latest | jq '.block.header.height | tonumber')
UPGRADE_HEIGHT=$(($latest_height + 30))

update_proposal ".messages[0][\"plan\"][\"name\"]=\"$NEW_VERSION\""
update_proposal ".messages[0][\"plan\"][\"height\"]=\"$UPGRADE_HEIGHT\""

$BINARY tx gov submit-proposal $CONFIG_DIR/proposals/proposal.json --from $VAL_1 --chain-id test-1 --home $DATA_DIR/test-1 --node tcp://localhost:16657 --keyring-backend test -y
sleep 2
$BINARY tx gov vote 2 yes --from $VAL_1 --chain-id test-1 --home $DATA_DIR/test-1 --node tcp://localhost:16657 --keyring-backend test -y

# sleep to wait til the proposal passes
echo "Sleep til the proposal passes..."

# Check if latest height is less than the upgrade height
latest_height=$(curl --no-progress-meter http://localhost:1316/cosmos/base/tendermint/v1beta1/blocks/latest | jq '.block.header.height | tonumber')
while [ $latest_height -lt $UPGRADE_HEIGHT ]; do
    sleep 5
    ((latest_height = $(curl --no-progress-meter http://localhost:1316/cosmos/base/tendermint/v1beta1/blocks/latest | jq '.block.header.height | tonumber')))
    echo $latest_height
done

# kill all processes
pkill $BINARY

# install new binary for the upgrade
echo "install new binary"
# clone or pull latest repo
if ! [ -d "$PWD/../orai-050" ]; then
    git clone https://github.com/oraichain/wasmd.git $PWD/../orai-050
fi
CUR_DIR=$PWD && cd $PWD/../orai-050 && git checkout $NEW_VERSION && go mod tidy && GOTOOLCHAIN=$GO_VERSION make build && cd $CUR_DIR

# re-run the nodes
bash $CONFIG_DIR/chains/start-single.sh
# start node 2 using different binary
CHAINID=test-2 BINARY=$OLD_BINARY bash $CONFIG_DIR/chains/start-single.sh

sleep 5

echo "Try to create channel but it doesn't work"
CHAIN1_PORT="icacontroller-$WALLET_2"
CHAIN2_PORT="icahost"
# create hermes channel
hermes create channel \
    --order ordered \
    --a-chain $CHAIN_ID1 \
    --a-connection $CONNECTION_ID \
    --a-port $CHAIN1_PORT \
    --b-port $CHAIN2_PORT

sleep 5

################################ upgrade again to v0502
NEW_VERSION="v0.50.2"
latest_height=$(curl --no-progress-meter http://localhost:1316/cosmos/base/tendermint/v1beta1/blocks/latest | jq '.block.header.height | tonumber')
UPGRADE_HEIGHT=$(($latest_height + 30))

update_proposal ".messages[0][\"plan\"][\"name\"]=\"$NEW_VERSION\""
update_proposal ".messages[0][\"plan\"][\"height\"]=\"$UPGRADE_HEIGHT\""

$BINARY tx gov submit-proposal $CONFIG_DIR/proposals/proposal.json --from $VAL_1 --chain-id test-1 --home $DATA_DIR/test-1 --node tcp://localhost:16657 --keyring-backend test -y
sleep 2
$BINARY tx gov vote 3 yes --from $VAL_1 --chain-id test-1 --home $DATA_DIR/test-1 --node tcp://localhost:16657 --keyring-backend test -y

# sleep to wait til the proposal passes
echo "Sleep til the proposal passes..."

# Check if latest height is less than the upgrade height
latest_height=$(curl --no-progress-meter http://localhost:1316/cosmos/base/tendermint/v1beta1/blocks/latest | jq '.block.header.height | tonumber')
while [ $latest_height -lt $UPGRADE_HEIGHT ]; do
    sleep 5
    ((latest_height = $(curl --no-progress-meter http://localhost:1316/cosmos/base/tendermint/v1beta1/blocks/latest | jq '.block.header.height | tonumber')))
    echo $latest_height
done

# kill all processes
pkill $BINARY

# install new binary for the upgrade
make build

# re-run the nodes
bash $CONFIG_DIR/chains/start-single.sh
# start node 2 using different binary
CHAINID=test-2 BINARY=$OLD_BINARY bash $CONFIG_DIR/chains/start-single.sh

sleep 5

echo "Re-try to create channel and it work"
CHAIN1_PORT="icacontroller-$WALLET_2"
CHAIN2_PORT="icahost"
# create hermes channel
hermes create channel \
    --order ordered \
    --a-chain $CHAIN_ID1 \
    --a-connection $CONNECTION_ID \
    --a-port $CHAIN1_PORT \
    --b-port $CHAIN2_PORT
