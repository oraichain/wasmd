#!/bin/bash
set -ux

BINARY=oraid
OLD_BINARY=oraid-0424

CONFIG_DIR="$PWD/scripts/ica-migration-tests"
DATA_DIR="$PWD/data"
GO_VERSION=$(go version | awk '{print $3}')

rm -rf data/
pkill oraid
pkill rly

# clone or pull latest repo
if ! [ -d "$PWD/../orai-0424" ]; then
  git clone --branch v0.42.4 --single-branch https://github.com/oraichain/orai.git $PWD/../orai-0424
fi

CUR_DIR=$PWD && cd $PWD/../orai-0424 && go mod tidy && GOTOOLCHAIN=go1.21.4 make install && cp $GOPATH/bin/oraid $GOPATH/bin/$OLD_BINARY && cd $CUR_DIR

echo "Initializing both blockchains..."
bash $CONFIG_DIR/init.sh
bash $CONFIG_DIR/start.sh
# wait for nodes to start 
echo "Initializing relayer..."
bash $CONFIG_DIR/go-rly-config/rly-init.sh

# start relayer
bash $CONFIG_DIR/go-rly-config/rly-start.sh > $DATA_DIR/relayer/rly.log 2>&1 &

# wait for rly to be up
sleep 5

# # Store the following account addresses within the current shell env
export WALLET_1=$($BINARY keys show wallet1 -a --keyring-backend test --home $DATA_DIR/test-1) && echo $WALLET_1;
export WALLET_2=$($BINARY keys show wallet2 -a --keyring-backend test --home $DATA_DIR/test-1) && echo $WALLET_2;
export VAL_1=$($BINARY keys show val1 -a --keyring-backend test --home $DATA_DIR/test-1) && echo $VAL_1;
export WALLET_3=$($BINARY keys show wallet3 -a --keyring-backend test --home $DATA_DIR/test-2) && echo $WALLET_3;
export WALLET_4=$($BINARY keys show wallet4 -a --keyring-backend test --home $DATA_DIR/test-2) && echo $WALLET_4;

# register ICA account
# Register an interchain account on behalf of WALLET_1 where chain test-2 is the interchain accounts host
$BINARY tx intertx register --from $WALLET_1 --connection-id connection-0 --chain-id test-1 --home $DATA_DIR/test-1 --node tcp://localhost:16657 --keyring-backend test -y -b block

# wait for rly to craete new interchain acc
sleep 5

# Query the address of the interchain account
$BINARY query intertx interchainaccounts connection-0 $WALLET_1 --home $DATA_DIR/test-1 --node tcp://localhost:16657

# Store the interchain account address by parsing the query result: cosmos1hd0f4u7zgptymmrn55h3hy20jv2u0ctdpq23cpe8m9pas8kzd87smtf8al
export ICA_ADDR=$($BINARY query intertx interchainaccounts connection-0 $WALLET_1 --home $DATA_DIR/test-1 --node tcp://localhost:16657 -o json | jq -r '.interchain_account_address') && echo $ICA_ADDR

# fund interchain account wallet
# Query the interchain account balance on the host chain. It should be empty.
$BINARY q bank balances $ICA_ADDR --chain-id test-2 --node tcp://localhost:26657

# Send funds to the interchain account.
$BINARY tx send $WALLET_3 $ICA_ADDR 10000orai --chain-id test-2 --home $DATA_DIR/test-2 --node tcp://localhost:26657 --keyring-backend test -y -b block

# Query the balance once again and observe the changes
$BINARY q bank balances $ICA_ADDR --chain-id test-2 --node tcp://localhost:26657

######### Test interchain txs
# Submit a bank send tx using the interchain account via ibc
$BINARY tx intertx submit \
"{
    \"@type\":\"/cosmos.bank.v1beta1.MsgSend\",
    \"from_address\":\"$ICA_ADDR\",
    \"to_address\":\"orai1knzg7jdc49ghnc2pkqg6vks8ccsk6efzfgv6gv\",
    \"amount\": [
        {
            \"denom\": \"orai\",
            \"amount\": \"1000\"
        }
    ]
}" --connection-id connection-0 --from $WALLET_1 --chain-id test-1 --home $DATA_DIR/test-1 --node tcp://localhost:16657 --keyring-backend test -y -b block

# Wait until the relayer has relayed the packet
sleep 5

# Query the interchain account balance on the host chain
amount=$($BINARY q bank balances $ICA_ADDR --chain-id test-2 --node tcp://localhost:26657 --denom orai --output json | jq '.amount | tonumber')

if [ $amount -ne 9000 ] ; then
  echo "ICA MsgSend Failed: $*"
  exit 1
fi

################################ upgrade chain
UPGRADE_HEIGHT=${UPGRADE_HEIGHT:-65}
NEW_VERSION=${NEW_VERSION:-"v0.50.0"}
$BINARY tx gov submit-proposal software-upgrade $NEW_VERSION --title "foobar" --description "foobar"  --from $WALLET_1 --upgrade-height $UPGRADE_HEIGHT --upgrade-info "x" --deposit 10000000orai --chain-id test-1 --home $DATA_DIR/test-1 --node tcp://localhost:16657 --keyring-backend test -y -b block
$BINARY tx gov vote 1 yes --from $VAL_1 --chain-id test-1 --home $DATA_DIR/test-1 --node tcp://localhost:16657 --keyring-backend test -y -b block

# sleep to wait til the proposal passes
echo "Sleep til the proposal passes..."

# Check if latest height is less than the upgrade height
latest_height=$(curl --no-progress-meter http://localhost:1316/cosmos/base/tendermint/v1beta1/blocks/latest | jq '.block.header.height | tonumber')
while [ $latest_height -lt $UPGRADE_HEIGHT ];
do
   sleep 5
   ((latest_height=$(curl --no-progress-meter http://localhost:1316/cosmos/base/tendermint/v1beta1/blocks/latest | jq '.block.header.height | tonumber')))
   echo $latest_height
done

# kill all processes
pkill $BINARY

# install new binary for the upgrade
echo "install new binary"
GOTOOLCHAIN=$GO_VERSION make build

# re-run the nodes
bash $CONFIG_DIR/start-single.sh
# start node 2 using different binary
CHAINID=test-2 BINARY=$OLD_BINARY bash $CONFIG_DIR/start-single.sh

sleep 5

################################ upgrade again to v0501
UPGRADE_HEIGHT=${UPGRADE_HEIGHT:-100}
NEW_VERSION=${NEW_VERSION:-"v0.50.1"}
$BINARY tx gov submit-proposal $CONFIG_DIR/proposal.json --from $VAL_1 --chain-id test-1 --home $DATA_DIR/test-1 --node tcp://localhost:16657 --keyring-backend test -y
sleep 2
$BINARY tx gov vote 2 yes --from $VAL_1 --chain-id test-1 --home $DATA_DIR/test-1 --node tcp://localhost:16657 --keyring-backend test -y

# sleep to wait til the proposal passes
echo "Sleep til the proposal passes..."

# Check if latest height is less than the upgrade height
latest_height=$(curl --no-progress-meter http://localhost:1316/cosmos/base/tendermint/v1beta1/blocks/latest | jq '.block.header.height | tonumber')
while [ $latest_height -lt $UPGRADE_HEIGHT ];
do
   sleep 5
   ((latest_height=$(curl --no-progress-meter http://localhost:1316/cosmos/base/tendermint/v1beta1/blocks/latest | jq '.block.header.height | tonumber')))
   echo $latest_height
done

# kill all processes
pkill $BINARY

# install new binary for the upgrade
echo "install new binary"
GOTOOLCHAIN=$GO_VERSION make build

# re-run the nodes
bash $CONFIG_DIR/start-single.sh
# start node 2 using different binary
CHAINID=test-2 BINARY=$OLD_BINARY bash $CONFIG_DIR/start-single.sh

sleep 5

# gen new ica message
oraid tx ica host generate-packet-data \
"{
    \"@type\":\"/cosmos.bank.v1beta1.MsgSend\",
    \"from_address\":\"$ICA_ADDR\",
    \"to_address\":\"orai1knzg7jdc49ghnc2pkqg6vks8ccsk6efzfgv6gv\",
    \"amount\": [
        {
            \"denom\": \"orai\",
            \"amount\": \"1000\"
        }
    ]
}" --encoding proto3 > $CONFIG_DIR/ica-send.json

# Submit a bank send tx using the interchain account via ibc
$BINARY tx ica controller send-tx connection-0 $CONFIG_DIR/ica-send.json --from $WALLET_1 --chain-id test-1 --home $DATA_DIR/test-1 --node tcp://localhost:16657 --keyring-backend test -y

# wait for relayer to process
sleep 5

amount=$($OLD_BINARY q bank balances $ICA_ADDR --chain-id test-2 --node tcp://localhost:26657 --denom orai --output json | jq '.amount | tonumber')

if [ $amount -ne 8000 ] ; then
  echo "ICA MsgSend After Upgrade Failed: $*"
  exit 1
fi

amount=$($OLD_BINARY q bank balances "orai1knzg7jdc49ghnc2pkqg6vks8ccsk6efzfgv6gv" --chain-id test-2 --node tcp://localhost:26657 --denom orai --output json | jq '.amount | tonumber')

if [ $amount -ne 2000 ] ; then
  echo "ICA MsgSend After Upgrade Failed. Receiver does not match balance: $*"
  exit 1
fi