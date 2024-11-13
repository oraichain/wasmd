#!/bin/bash
set -ux

BINARY=oraid

CONFIG_DIR="$PWD/scripts/ica-migration-tests"
DATA_DIR="$PWD/data"

rm -rf data
pkill oraid
pkill rly

echo "Initializing both blockchains..."
bash $CONFIG_DIR/init.sh
bash $CONFIG_DIR/start.sh
echo "Initializing relayer..."
bash $CONFIG_DIR/go-rly-config/rly-init.sh

# start node
bash $CONFIG_DIR/start.sh
# start relayer
bash $CONFIG_DIR/go-rly-config/rly-start.sh > $DATA_DIR/relayer/rly.log 2>&1 &

# wait for rly to be up
sleep 3

# Store the following account addresses within the current shell env
export WALLET_1=$($BINARY keys show wallet1 -a --keyring-backend test --home $DATA_DIR/test-1) && echo $WALLET_1;
export WALLET_2=$($BINARY keys show wallet2 -a --keyring-backend test --home $DATA_DIR/test-1) && echo $WALLET_2;
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
'{
    "@type":"/cosmos.bank.v1beta1.MsgSend",
    "from_address":"cosmos15ccshhmp0gsx29qpqq6g4zmltnnvgmyu9ueuadh9y2nc5zj0szls5gtddz",
    "to_address":"cosmos10h9stc5v6ntgeygf5xf945njqq5h32r53uquvw",
    "amount": [
        {
            "denom": "orai",
            "amount": "1000"
        }
    ]
}' --connection-id connection-0 --from $WALLET_1 --chain-id test-1 --home $DATA_DIR/test-1 --node tcp://localhost:16657 --keyring-backend test -y -b block

# Wait until the relayer has relayed the packet
sleep 5

Query the interchain account balance on the host chain
amount=$($BINARY q bank balances $ICA_ADDR --chain-id test-2 --node tcp://localhost:26657 --denom orai --output json | jq '.amount | tonumber')

if [ $amount -ne 10000 ] ; then
  echo "ICA MsgSend Failed: $*"
  exit 1
fi