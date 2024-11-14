#!/bin/sh
set -eo pipefail

DEFAULT_CHAIN_ID="localoraichain"
DEFAULT_VALIDATOR_MONIKER="validator"
DEFAULT_VALIDATOR_MNEMONIC="bottom loan skill merry east cradle onion journey palm apology verb edit desert impose absurd oil bubble sweet glove shallow size build burst effort"
DEFAULT_FAUCET_MNEMONIC="increase bread alpha rigid glide amused approve oblige print asset idea enact lawn proof unfold jeans rabbit audit return chuckle valve rather cactus great"
DEFAULT_RELAYER_MNEMONIC="black frequent sponsor nice claim rally hunt suit parent size stumble expire forest avocado mistake agree trend witness lounge shiver image smoke stool chicken"

# Override default values with environment variables
CHAIN_ID=${CHAIN_ID:-$DEFAULT_CHAIN_ID}
VALIDATOR_MONIKER=${VALIDATOR_MONIKER:-$DEFAULT_VALIDATOR_MONIKER}
VALIDATOR_MNEMONIC=${VALIDATOR_MNEMONIC:-$DEFAULT_VALIDATOR_MNEMONIC}
FAUCET_MNEMONIC=${FAUCET_MNEMONIC:-$DEFAULT_FAUCET_MNEMONIC}
RELAYER_MNEMONIC=${RELAYER_MNEMONIC:-$DEFAULT_RELAYER_MNEMONIC}

ORAICHAIN_HOME=$HOME/.oraid
CONFIG_FOLDER=$ORAICHAIN_HOME/config

install_prerequisites() {
    apk add jq
    apk add sed
}

update_genesis() {
    cat $CONFIG_FOLDER/genesis.json | jq "$1" >$CONFIG_FOLDER/tmp_genesis.json && mv $CONFIG_FOLDER/tmp_genesis.json $CONFIG_FOLDER/genesis.json
}

edit_genesis() {
    # change staking denom to orai
    update_genesis '.app_state["staking"]["params"]["bond_denom"]="orai"'
    # update staking genesis
    update_genesis '.app_state["staking"]["params"]["unbonding_time"]="240s"'
    # update crisis variable to orai
    update_genesis '.app_state["crisis"]["constant_fee"]["denom"]="orai"'
    # udpate gov genesis
    update_genesis '.app_state["gov"]["params"]["min_deposit"][0]["denom"]="orai"'
    update_genesis '.app_state["gov"]["params"]["expedited_min_deposit"][0]["denom"]="orai"'
    update_genesis '.app_state["gov"]["params"]["voting_period"]="6s"'
    update_genesis '.app_state["gov"]["params"]["expedited_voting_period"]="5.999999999s"'

    # update mint genesis
    update_genesis '.app_state["mint"]["params"]["mint_denom"]="orai"'
    update_genesis '.initial_height="1"'

}

add_genesis_accounts() {
    # Validator
    echo "⚖️ Add validator account"
    echo $VALIDATOR_MNEMONIC | oraid keys add $VALIDATOR_MONIKER --recover --keyring-backend=test --home $ORAICHAIN_HOME
    VALIDATOR_ACCOUNT=$(oraid keys show -a $VALIDATOR_MONIKER --keyring-backend test --home $ORAICHAIN_HOME)
    oraid genesis add-genesis-account $VALIDATOR_ACCOUNT 1000000000000orai --home $ORAICHAIN_HOME
    echo "🔍 Check initial height 1"
    jq '.initial_height' $CONFIG_FOLDER/genesis.json
    # Faucet
    echo "🚰 Add faucet account"
    echo $FAUCET_MNEMONIC | oraid keys add faucet --recover --keyring-backend=test --home $ORAICHAIN_HOME
    FAUCET_ACCOUNT=$(oraid keys show -a faucet --keyring-backend test --home $ORAICHAIN_HOME)
    oraid genesis add-genesis-account $FAUCET_ACCOUNT 100000000000orai --home $ORAICHAIN_HOME

    # Relayer
    echo "🔗 Add relayer account"
    echo $RELAYER_MNEMONIC | oraid keys add relayer --recover --keyring-backend=test --home $ORAICHAIN_HOME
    RELAYER_ACCOUNT=$(oraid keys show -a relayer --keyring-backend test --home $ORAICHAIN_HOME)
    oraid genesis add-genesis-account $RELAYER_ACCOUNT 1000000000orai --home $ORAICHAIN_HOME

    oraid genesis gentx $VALIDATOR_MONIKER 500000000orai --keyring-backend=test --chain-id=$CHAIN_ID --home $ORAICHAIN_HOME
    oraid genesis collect-gentxs --home $ORAICHAIN_HOME

}

edit_config() {
    pruning="custom"
    pruning_keep_recent="5"
    pruning_keep_every="10"
    pruning_interval="10000"
    sed -i -e "s%^pruning *=.*%pruning = \"$pruning\"%; " $CONFIG_FOLDER/app.toml
    sed -i -e "s%^pruning-keep-recent *=.*%pruning-keep-recent = \"$pruning_keep_recent\"%; " $CONFIG_FOLDER/app.toml
    sed -i -e "s%^pruning-interval *=.*%pruning-interval = \"$pruning_interval\"%; " $CONFIG_FOLDER/app.toml
    snapshot_interval="10"
    snapshot_keep_recent="2"
    sed -i -e "s%^snapshot-interval *=.*%snapshot-interval = \"$snapshot_interval\"%; " $CONFIG_FOLDER/app.toml
    sed -i -e "s%^snapshot-keep-recent *=.*%snapshot-keep-recent = \"$snapshot_keep_recent\"%; " $CONFIG_FOLDER/app.toml

    sed -i -E 's|allow_duplicate_ip = false|allow_duplicate_ip = true|g' $CONFIG_FOLDER/config.toml
    sed -i -e "s%^timeout_broadcast_tx_commit *=.*%timeout_broadcast_tx_commit = \"60s\"%; " $CONFIG_FOLDER/config.toml

    # Expose the rpc
    sed -i -E 's|tcp://127.0.0.1:26657|tcp://0.0.0.0:26657|g' $CONFIG_FOLDER/config.toml
    sed -i -E 's|localhost:9090|0.0.0.0:9090|g' $CONFIG_FOLDER/app.toml

    # Expose the grpc
    # dasel put -t string -f $CONFIG_FOLDER/app.toml -v "0.0.0.0:9090" '.grpc.address'
}

if [[ ! -d $CONFIG_FOLDER ]]; then
    install_prerequisites
    echo "🧪 Creating Oraichain home for $VALIDATOR_MONIKER"
    echo $VALIDATOR_MNEMONIC | oraid init -o --chain-id=$CHAIN_ID --home $ORAICHAIN_HOME --recover $VALIDATOR_MONIKER
    edit_genesis
    add_genesis_accounts
    edit_config
    update_genesis '.initial_height="1"'
fi

echo "🏁 Starting $CHAIN_ID..."
oraid start --home $ORAICHAIN_HOME --log_level=debug
