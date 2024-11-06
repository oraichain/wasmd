# ABCI State Streaming

The `BaseApp` package contains the interface for a [ABCIListener](https://github.com/cosmos/cosmos-sdk/blob/main/baseapp/streaming.go)
service used to write state changes out from individual KVStores to external systems,
as described in [ADR-038](https://github.com/cosmos/cosmos-sdk/blob/main/docs/architecture/adr-038-state-listening.md).

Specific `ABCIListener` service implementations are written and loaded as [hashicorp/go-plugin](https://github.com/hashicorp/go-plugin).

Oraichain Labs leverages the power of Cosmos SDK's state streaming to index custom module's data into Postgres, while emitting events to other sources so applications can subscribe

## Configuration

Update the streaming section in `app.toml` to enable ABCI state streaming

```toml
# Streaming allows nodes to stream state to external systems
[streaming]

# streaming.abci specifies the configuration for the ABCI Listener streaming service
[streaming.abci]

# List of kv store keys to stream out via gRPC
# Set to ["*"] to expose all keys.
keys = ["*"]

# The plugin name used for streaming via gRPC
plugin = "abci"

# stop-node-on-err specifies whether to stop the node when the plugin has problems
stop-node-on-err = false

# streaming.wasm specifies the configuration for the ABCI Listener streaming service, for the wasm module
[streaming.wasm]

# The plugin name used for streaming via gRPC
plugin = "wasm"
```

Note that the ABCI plugin is a must-have. You can add additional plugins, but the `keys` and `stop-node-on-err` fields in `app.toml` only take values from the ABCI plugin.

## Build the plugin

In the base directory (wasmd/), run the following command to build the plugin:

```shell
# build the plugin
go build -o streaming/streaming streaming/streaming.go

# export env variable so the plugin can be seen by the node. Ref: https://github.com/oraichain/cosmos-sdk/blob/f503e9b2186f54e8480dd35e5033a03ebc8e8dac/baseapp/streaming.go#L35. The method initializes a new streaming plugin and runs it using an env variable path COSMOS_SDK_<plugin-name>
export COSMOS_SDK_ABCI="{path to}/streaming/streaming"

# build another plugin
go build -o streaming/streaming streaming/wasm_streaming.go

# export env variable so the plugin can be seen by the node
export COSMOS_SDK_WASM="{path to}/streaming/wasm_streaming"
```
