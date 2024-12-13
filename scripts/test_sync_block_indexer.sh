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
BANKTX_ARGS="$ARGS --chain-id $CHAINID --gas 200000 --fees 2orai --node http://localhost:26657 --yes"
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

jq '.initial_height="1"' $NODE_HOME/config/genesis.json >tmp.$$.json && mv tmp.$$.json $NODE_HOME/config/genesis.json

APP_TOML=$NODE_HOME/config/app.toml
CONFIG_TOML=$NODE_HOME/config/config.toml

# export cosmos sdk streaming plugin path
export COSMOS_SDK_ABCI="$PWD/streaming/streaming"
export REDPANDA_BROKERS="localhost:19092"
# build streaming plugin
go build -o $PWD/streaming/streaming $PWD/streaming/streaming.go

# add indexer info
sed -i '' -E "s%^indexer *=.*%indexer = \"null\"%; " $CONFIG_TOML
sed -i '' -E "s%^psql-conn *=.*%psql-conn = \"$PSQL_CONN\"%; " $CONFIG_TOML

# export PSQL conn and chain id
export PSQL_CONN="postgresql://admin:root@localhost:5432/node_indexer?sslmode=disable"
export CHAIN_ID=$CHAINID
export HOME_PATH=$NODE_HOME

# add indexer.toml file to enable indexer RPC
cp $PWD/scripts/indexer.toml $NODE_HOME/config

# clean old db
docker-compose -f $PWD/indexer/docker-compose.yml down -v
# start new db
docker-compose -f $PWD/indexer/docker-compose.yml up -d

# clean old redpanda
docker-compose -f $PWD/streaming/docker-compose.yml down -v
# start new redpanda
docker-compose -f $PWD/streaming/docker-compose.yml up -d

# sleep a bit for psql to be up
sleep 5

# run 20 first blocks without indexing
screen -S node_indexer -d -m oraid start $START_ARGS --halt-height 20

# send bank tx 
sleep 5
VALIDATOR_ADDRESS=$(oraid keys show $USER -a --keyring-backend test --home $NODE_HOME)
oraid tx bank send $VALIDATOR_ADDRESS orai1kzkf6gttxqar9yrkxfe34ye4vg5v4m588ew7c9 10000orai $BANKTX_ARGS

# sleep for node run pass 20 first blocks
sleep 20
# kill all processes
pkill oraid

# add streaming info
sed -i '' -E "s%^keys *=.*%keys = [\"*\"]%; " $APP_TOML
sed -i '' -E "s%^plugin *=.*%plugin = \"abci\"%; " $APP_TOML

# restart node
screen -S node_indexer -d -m oraid start $START_ARGS

sleep 5

# run test streaming + indexer
sh $PWD/scripts/test_streaming.sh

sleep 5

# sync from block 1 to block 20
go run $PWD/indexer/sync/main.go --archive-node http://localhost:26657 --start-block 1 --end-block 20

sleep 2
echo "Tests passed!"
bash $PWD/scripts/clean-multinode-local-testnet.sh