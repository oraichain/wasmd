#!/bin/bash
set -ux

CHAINID=${CHAINID:-testing}
USER=${USER:-tupt}
MONIKER=${MONIKER:-node001}
HIDE_LOGS="/dev/null"
# PASSWORD=${PASSWORD:-$1}
NODE_HOME="$PWD/.oraid"
ARGS="--keyring-backend test --home $NODE_HOME"
START_ARGS="--json-rpc.address="0.0.0.0:8545" --json-rpc.ws-address="0.0.0.0:8546" --json-rpc.api="eth,web3,net,txpool,debug" --json-rpc.enable --home $NODE_HOME"
PSQL_CONN=${PSQL_CONN:-"postgresql://admin:root@localhost:5432/node_indexer?sslmode=disable"}

rm -rf $NODE_HOME

oraid init --chain-id "$CHAINID" "$MONIKER" --home $NODE_HOME >$HIDE_LOGS

oraid keys add $USER $ARGS 2>&1 | tee account.txt
oraid keys add $USER-eth $ARGS --eth 2>&1 | tee account-eth.txt
oraid keys unsafe-export-eth-key $USER-eth $ARGS 2>&1 | tee priv-eth.txt

# hardcode the validator account for this instance
oraid genesis add-genesis-account $USER "100000000000000orai" $ARGS
oraid genesis add-genesis-account $USER-eth "100000000000000orai" $ARGS
oraid genesis add-genesis-account orai1kzkf6gttxqar9yrkxfe34ye4vg5v4m588ew7c9 "100000000000000orai" $ARGS

# submit a genesis validator tx
# Workraround for https://github.com/cosmos/cosmos-sdk/issues/8251
oraid genesis gentx $USER "250000000orai" --chain-id="$CHAINID" -y $ARGS >$HIDE_LOGS

oraid genesis collect-gentxs --home $NODE_HOME >$HIDE_LOGS

jq '.initial_height="1"' $NODE_HOME/config/genesis.json > tmp.$$.json && mv tmp.$$.json $NODE_HOME/config/genesis.json

APP_TOML=$NODE_HOME/config/app.toml
CONFIG_TOML=$NODE_HOME/config/config.toml
# add streaming info
sed -i '' -E "s%^keys *=.*%keys = [\"*\"]%; " $APP_TOML
sed -i '' -E "s%^plugin *=.*%plugin = \"abci\"%; " $APP_TOML
# export cosmos sdk streaming plugin path
export COSMOS_SDK_ABCI="$PWD/streaming/streaming"
# build streaming plugin
go build -o $PWD/streaming/streaming $PWD/streaming/streaming.go

# add indexer info
sed -i '' -E "s%^indexer *=.*%indexer = \"psql\"%; " $CONFIG_TOML
sed -i '' -E "s%^psql-conn *=.*%psql-conn = \"$PSQL_CONN\"%; " $CONFIG_TOML

# export PSQL conn and chain id
export PSQL_CONN="postgresql://admin:root@localhost:5432/node_indexer?sslmode=disable"
export CHAIN_ID=$CHAINID

# add indexer.toml file to enable indexer RPC
cp $PWD/indexer.toml $NODE_HOME/config

# clean old db
docker-compose -f $PWD/indexer/docker-compose.yml down -v
# start new db
docker-compose -f $PWD/indexer/docker-compose.yml up -d

# sleep a bit for psql to be up
sleep 5

oraid start $START_ARGS