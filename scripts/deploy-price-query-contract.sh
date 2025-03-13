WASM_PATH=${WASM_PATH:-"scripts/wasm_file/price-query-local.wasm"}
ARGS="--chain-id testing -y --keyring-backend test --fees 200orai --gas auto --gas-adjustment 1.5"
VALIDATOR_HOME=${VALIDATOR_HOME:-"$HOME/.oraid/validator1"}
WASM_MOCK_ORAIDEX_PATH=${WASM_MOCK_ORAIDEX_PATH:-"scripts/wasm_file/mock-oraidex-v3.wasm"}

# Deploy new mock oraidex contract
store_ret=$(oraid tx wasm store $WASM_MOCK_ORAIDEX_PATH --from validator1 --home $VALIDATOR_HOME $ARGS --output json)
tx_hash=$(echo $store_ret | jq -r '.txhash')

# Wait for transaction to be committed
sleep 2

# Get code_id using tx_hash
code_id=$(oraid query tx $tx_hash --output json | jq -r '.events[] | select(.type == "store_code") | .attributes[] | select(.key == "code_id") | .value')

# Instantiate contract
INSTANTIATE_MSG='{}'
instantiate_ret=$(oraid tx wasm instantiate $code_id "$INSTANTIATE_MSG" --label 'testing' --from validator1 --home $VALIDATOR_HOME --admin $(oraid keys show validator1 --keyring-backend test --home $VALIDATOR_HOME -a) $ARGS --output json)
instantiate_tx_hash=$(echo $instantiate_ret | jq -r '.txhash')

# Wait for transaction to be committed
sleep 2

# Get contract address
mock_oraidex_address=$(oraid query tx $instantiate_tx_hash --output json | jq -r '.events[] | select(.type == "instantiate") | .attributes[] | select(.key == "_contract_address") | .value')

sleep 2

# query pool
POOL='{"pool": {"token_0": "factory/orai1wuvhex9xqs3r539mvc6mtm7n20fcj3qr2m0y9khx6n5vtlngfzes3k0rq9/DYeTA4ZQhEwoJ5imjq1Q3zgwfTgkh4WmdfFHAq3jLrv3", "token_1": "orai", "fee_tier": {"fee": 3000000000, "tick_spacing": 100}}}'
oraid query wasm contract-state smart $mock_oraidex_address "$POOL" --output json

# Deploy price query contract
store_ret=$(oraid tx wasm store $WASM_PATH --from validator1 --home $VALIDATOR_HOME $ARGS --output json)
tx_hash=$(echo $store_ret | jq -r '.txhash')

# Wait for transaction to be committed
sleep 2

# Get code_id using tx_hash
code_id=$(oraid query tx $tx_hash --output json | jq -r '.events[] | select(.type == "store_code") | .attributes[] | select(.key == "code_id") | .value')

# Instantiate contract
INSTANTIATE_MSG=$(jq -n --arg base_token "orai" --arg oraidex_v3_addr "$mock_oraidex_address" \
    '{"base_token": $base_token, "oraidex_v3_addr": $oraidex_v3_addr}')
instantiate_ret=$(oraid tx wasm instantiate $code_id "$INSTANTIATE_MSG" --label 'testing' --from validator1 --home $VALIDATOR_HOME --admin $(oraid keys show validator1 --keyring-backend test --home $VALIDATOR_HOME -a) $ARGS --output json)
instantiate_tx_hash=$(echo $instantiate_ret | jq -r '.txhash')

# Wait for transaction to be committed
sleep 2

# Get contract address
contract_address=$(oraid query tx $instantiate_tx_hash --output json | jq -r '.events[] | select(.type == "instantiate") | .attributes[] | select(.key == "_contract_address") | .value')

sleep 2

# add pool list
ADD_POOL='{"add_pool": {"quote_token": "factory/orai1wuvhex9xqs3r539mvc6mtm7n20fcj3qr2m0y9khx6n5vtlngfzes3k0rq9/DYeTA4ZQhEwoJ5imjq1Q3zgwfTgkh4WmdfFHAq3jLrv3","fee_tier": {"fee": 3000000000, "tick_spacing": 100}}}'
oraid tx wasm execute $contract_address "$ADD_POOL" --from validator1 --home $VALIDATOR_HOME $ARGS --output json

sleep 2

# Query sqrt price of usdai
GET_SQRT_PRICE='{"get_sqrt_price": {"quote_token": "factory/orai1wuvhex9xqs3r539mvc6mtm7n20fcj3qr2m0y9khx6n5vtlngfzes3k0rq9/DYeTA4ZQhEwoJ5imjq1Q3zgwfTgkh4WmdfFHAq3jLrv3"}}'
oraid query wasm contract-state smart $contract_address "$GET_SQRT_PRICE" --output json

# add usdc to pool
ADD_POOL='{"add_pool": {"quote_token": "usdc","fee_tier": {"fee": 5000000000, "tick_spacing": 60}}}'
oraid tx wasm execute $contract_address "$ADD_POOL" --from validator1 --home $VALIDATOR_HOME $ARGS --output json

sleep 2
# Query sqrt price of usdt => expect ERROR
# GET_SQRT_PRICE='{"get_sqrt_price": {"quote_token": "usdt"}}'
# oraid query wasm contract-state smart $contract_address "$GET_SQRT_PRICE" --output json

# Query sqrt price of usdc
GET_SQRT_PRICE='{"get_sqrt_price": {"quote_token": "usdc"}}'
oraid query wasm contract-state smart $contract_address "$GET_SQRT_PRICE" --output json
