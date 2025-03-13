WASM_PATH=${WASM_PATH:-"scripts/wasm_file/price-query-local-new.wasm"}
ARGS="--chain-id testing -y --keyring-backend test --fees 200orai --gas auto --gas-adjustment 1.5"
VALIDATOR_HOME=${VALIDATOR_HOME:-"$HOME/.oraid/validator1"}

# Deploy new contract
store_ret=$(oraid tx wasm store $WASM_PATH --from validator1 --home $VALIDATOR_HOME $ARGS --output json)
tx_hash=$(echo $store_ret | jq -r '.txhash')

# Wait for transaction to be committed
sleep 2s

# Get code_id using tx_hash
code_id=$(oraid query tx $tx_hash --output json | jq -r '.events[] | select(.type == "store_code") | .attributes[] | select(.key == "code_id") | .value')

# Instantiate contract
INSTANTIATE_MSG='{"base_token": "orai", "oraidex_v3_addr": "orai1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrq3e4sxg"}'
instantiate_ret=$(oraid tx wasm instantiate $code_id "$INSTANTIATE_MSG" --label 'testing' --from validator1 --home $VALIDATOR_HOME --admin $(oraid keys show validator1 --keyring-backend test --home $VALIDATOR_HOME -a) $ARGS --output json)
instantiate_tx_hash=$(echo $instantiate_ret | jq -r '.txhash')

# Wait for transaction to be committed
sleep 2s

# Get contract address
contract_address=$(oraid query tx $instantiate_tx_hash --output json | jq -r '.events[] | select(.type == "instantiate") | .attributes[] | select(.key == "_contract_address") | .value')

sleep 2s

# add pool list
ADD_POOL='{"add_pool": {"quote_token": "factory/orai1wuvhex9xqs3r539mvc6mtm7n20fcj3qr2m0y9khx6n5vtlngfzes3k0rq9/DYeTA4ZQhEwoJ5imjq1Q3zgwfTgkh4WmdfFHAq3jLrv3","fee_tier": {"fee": 3000000000, "tick_spacing": 100}}}'
oraid tx wasm execute $contract_address "$ADD_POOL" --from validator1 --home $VALIDATOR_HOME $ARGS --output json

sleep 2s

# Query sqrt price of usdai
GET_SQRT_PRICE='{"get_sqrt_price": {"quote_token": "factory/orai1wuvhex9xqs3r539mvc6mtm7n20fcj3qr2m0y9khx6n5vtlngfzes3k0rq9/DYeTA4ZQhEwoJ5imjq1Q3zgwfTgkh4WmdfFHAq3jLrv3"}}'
oraid query wasm contract-state smart $contract_address "$GET_SQRT_PRICE" --output json

# add new token denom
ADD_POOL='{"add_pool": {"quote_token": "usdc","fee_tier": {"fee": 5000000000, "tick_spacing": 60}}}'
oraid tx wasm execute $contract_address "$ADD_POOL" --from validator1 --home $VALIDATOR_HOME $ARGS --output json

# Query sqrt price of usdt => expect ERROR
GET_SQRT_PRICE='{"get_sqrt_price": {"quote_token": "usdt"}}'
oraid query wasm contract-state smart $contract_address "$GET_SQRT_PRICE" --output json

# Query sqrt price of usdc
GET_SQRT_PRICE='{"get_sqrt_price": {"quote_token": "usdc"}}'
oraid query wasm contract-state smart $contract_address "$GET_SQRT_PRICE" --output json
