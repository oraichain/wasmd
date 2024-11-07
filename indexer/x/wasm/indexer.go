package wasm

import (
	"context"

	"github.com/CosmWasm/wasmd/app/params"
	"github.com/CosmWasm/wasmd/indexer"
	"github.com/CosmWasm/wasmd/x/wasm/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/state/indexer/sink/psql"
)

// EventSink is an indexer backend providing the tx/block index services.  This
// implementation stores records in a PostgreSQL database using the schema
// defined in state/indexer/sink/psql/schema.sql.
type WasmEventSink struct {
	es             *psql.EventSink
	encodingConfig params.EncodingConfig
}

var _ indexer.ModuleEventSinkIndexer = (*WasmEventSink)(nil)

func NewWasmEventSinkIndexer(es *psql.EventSink, encodingConfig params.EncodingConfig) *WasmEventSink {
	return &WasmEventSink{es: es, encodingConfig: encodingConfig}
}

func (cs *WasmEventSink) InsertModuleEvents(req *abci.RequestFinalizeBlock, res *abci.ResponseFinalizeBlock) error {
	return nil
}

func (cs *WasmEventSink) EmitModuleEvents(ctx context.Context, req *abci.RequestFinalizeBlock, res *abci.ResponseFinalizeBlock) error {
	return nil
}

func (cs *WasmEventSink) ModuleName() string {
	return types.ModuleName
}

func (cs *WasmEventSink) EventSink() *psql.EventSink {
	return cs.es
}

func (cs *WasmEventSink) EncodingConfig() params.EncodingConfig {
	return cs.encodingConfig
}
