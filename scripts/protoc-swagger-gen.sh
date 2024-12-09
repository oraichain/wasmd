#!/usr/bin/env bash

# npm install -g swagger2openapi -> convert swagger 2.0 to openapi 3.0
# brew install buf - install buf for mac
# npm install -g swagger-combine -> combine swagger json

set -eo pipefail

mkdir -p ./tmp-swagger-gen
cd proto
COSMOS_SDK_DIR=${COSMOS_SDK_DIR:-$(go list -f "{{ .Dir }}" -m github.com/cosmos/cosmos-sdk)}
IBC_DIR=${IBC_DIR:-$(go list -f "{{ .Dir }}" -m github.com/cosmos/ibc-go/v8)}
ETHERMINT_DIR=${ETHERMINT_DIR:-$(go list -f "{{ .Dir }}" -m github.com/evmos/ethermint)}
proto_dirs=$(find ./cosmwasm $COSMOS_SDK_DIR/proto/cosmos $IBC_DIR/proto/ibc $IBC_DIR/proto/capability $ETHERMINT_DIR/proto/ethermint -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
for dir in $proto_dirs; do
  # generate swagger files (filter query files)
  query_file=$(find "${dir}" -maxdepth 1 \( -name 'query.proto' -o -name 'service.proto' \))
  if [[ ! -z "$query_file" ]]; then
    buf generate --template buf.gen.swagger.yml $query_file
  fi
done

cd ..
# combine swagger files
# uses nodejs package `swagger-combine`.
# all the individual swagger files need to be configured in `config.json` for merging
swagger-combine ./client/docs/swagger-ui/config.json -o ./client/docs/swagger-ui/swagger.yaml -f yaml --continueOnConflictingPaths true --includeDefinitions true

# convert swagger yaml to openapi
swagger2openapi --yaml --outfile ./client/docs/swagger-ui/openapi.yaml ./client/docs/swagger-ui/swagger.yaml

# clean swagger files
rm -rf ./tmp-swagger-gen
