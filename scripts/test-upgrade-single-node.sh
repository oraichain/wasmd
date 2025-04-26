#!/bin/sh
set -ux

VERSION=${VERSION:-"v0.50.11"}
WALLET="oraichain"
RPC="http://localhost:26657"
ARGS="--from $WALLET --keyring-backend test --home $PWD/.oraid --chain-id testing --fees 10000orai --gas auto --gas-adjustment 1.5 --node $RPC -y"

update_proposal() {
    cat $PWD/scripts/json/proposal.json | jq "$1" >$PWD/scripts/json/temp_proposal.json && mv $PWD/scripts/json/temp_proposal.json $PWD/scripts/json/proposal.json
}

# update authority proposal.json
MODULE_ACCOUNT=$(oraid query auth module-account gov --output json --node $RPC | jq '.account.value.address')
update_proposal ".messages[0][\"authority\"]=$MODULE_ACCOUNT"
update_proposal ".messages[0][\"plan\"][\"name\"]=\"$VERSION\""

# height = 5 days + 14 hours, each second has 1.45 block
HEIGHT=100
UPGRADE_HEIGHT=$(curl --no-progress-meter $RPC/block | jq '.result.block.header.height | tonumber')
UPGRADE_HEIGHT=$((UPGRADE_HEIGHT + $HEIGHT))
update_proposal ".messages[0][\"plan\"][\"height\"]=\"$UPGRADE_HEIGHT\""

oraid tx gov submit-proposal $PWD/scripts/json/proposal.json $ARGS

sleep 2

oraid tx gov vote 1 yes $ARGS
