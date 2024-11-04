package main

import (
	"context"

	"github.com/CosmWasm/wasmd/indexer"
	"github.com/CosmWasm/wasmd/indexer/sinkreader"
	"github.com/CosmWasm/wasmd/indexer/x/wasm"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/state/indexer/sink/psql"
	"github.com/hashicorp/go-plugin"

	streamingabci "cosmossdk.io/store/streaming/abci"
	store "cosmossdk.io/store/types"

	// "database/sql"

	// Register the Postgres database driver.
	_ "github.com/lib/pq"
)

// ModsStreamingPlugin is the implementation of the ABCIListener interface
// For Go plugins this is all that is required to process data sent over gRPC.
type ModsStreamingPlugin struct {
	indexerManager *indexer.IndexerManager
	es             *psql.EventSink
	reader         sinkreader.EventSinkReader
}

func (p *ModsStreamingPlugin) initStreamIndexerConn() {
	// init psql conn for indexing complex ops if is nil
	if p.es == nil {
		psqlConn, chainID := p.reader.ReadEventSinkInfo()
		es, err := psql.NewEventSink(psqlConn, chainID)
		if err != nil {
			panic(err)
		}
		p.es = es
	}
}

func (p *ModsStreamingPlugin) initIndexerManager() {
	if p.indexerManager == nil {
		p.indexerManager = indexer.NewIndexerManager(wasm.NewWasmEventSinkIndexer(p.es))
	}
}

func (a *ModsStreamingPlugin) ListenFinalizeBlock(ctx context.Context, req abci.RequestFinalizeBlock, res abci.ResponseFinalizeBlock) error {
	a.initStreamIndexerConn()
	a.initIndexerManager()
	for _, indexer := range a.indexerManager.Modules {
		err := indexer.InsertModuleEvents(&req, &res)
		if err != nil {
			return err
		}
		err = indexer.EmitModuleEvents(&req, &res)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *ModsStreamingPlugin) ListenCommit(ctx context.Context, res abci.ResponseCommit, changeSet []*store.StoreKVPair) error {
	// process block commit messages (i.e: sent to external system)
	return nil
}

func main() {

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: streamingabci.Handshake,
		Plugins: map[string]plugin.Plugin{
			"abci": &streamingabci.ListenerGRPCPlugin{Impl: &ModsStreamingPlugin{reader: sinkreader.NewEventSinkReader()}},
		},

		// A non-nil value here enables gRPC serving for this streaming...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
