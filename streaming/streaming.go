package main

import (
	"context"

	"github.com/CosmWasm/wasmd/app/params"
	"github.com/CosmWasm/wasmd/indexer"
	"github.com/CosmWasm/wasmd/indexer/codec"
	"github.com/CosmWasm/wasmd/indexer/sinkreader"
	"github.com/CosmWasm/wasmd/indexer/x/tx"
	"github.com/CosmWasm/wasmd/indexer/x/wasm"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/state/indexer/sink/psql"
	"github.com/hashicorp/go-plugin"

	streamingabci "cosmossdk.io/store/streaming/abci"
	store "cosmossdk.io/store/types"

	_ "github.com/lib/pq"
)

var encodingConfig params.EncodingConfig

func init() {
	encodingConfig = codec.MakeEncodingConfig()
}

// ModsStreamingPlugin is the implementation of the ABCIListener interface
// For Go plugins this is all that is required to process data sent over gRPC.
type ModsStreamingPlugin struct {
	indexerManager *indexer.IndexerManager
	es             *psql.EventSink
	reader         sinkreader.EventSinkReader
}

func (p *ModsStreamingPlugin) initStreamIndexerConn() error {
	// init psql conn for indexing complex ops if is nil
	if p.es == nil {
		psqlConn, chainID, err := p.reader.ReadEventSinkInfo()
		if err != nil {
			return err
		}
		es, err := psql.NewEventSink(psqlConn, chainID)
		if err != nil {
			panic(err)
		}
		p.es = es
	}
	return nil
}

func (p *ModsStreamingPlugin) initIndexerManager() {
	if p.indexerManager == nil {
		// orders matter! the tx indexer must always be at the top to insert tx requests & block events into postgres
		p.indexerManager = indexer.NewIndexerManager(
			tx.NewTxEventSinkIndexer(p.es, encodingConfig),
			wasm.NewWasmEventSinkIndexer(p.es, encodingConfig),
		)
	}
}

func (a *ModsStreamingPlugin) ListenFinalizeBlock(ctx context.Context, req abci.RequestFinalizeBlock, res abci.ResponseFinalizeBlock) error {
	if err := a.initStreamIndexerConn(); err != nil {
		return err
	}
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
