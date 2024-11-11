package wasm

import (
	"github.com/CosmWasm/wasmd/app/params"
	"github.com/CosmWasm/wasmd/indexer"
	"github.com/CosmWasm/wasmd/streaming/redpanda"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/state/indexer/sink/psql"
)

// EventSink is an indexer backend providing the tx/block index services.  This
// implementation stores records in a PostgreSQL database using the schema
// defined in state/indexer/sink/psql/schema.sql.
type WasmEventSink struct {
	es             *psql.EventSink
	encodingConfig params.EncodingConfig
	ri             *redpanda.RedpandaInfo
}

var _ indexer.ModuleEventSinkIndexer = (*WasmEventSink)(nil)

func NewWasmEventSinkIndexer(es *psql.EventSink, encodingConfig params.EncodingConfig) *WasmEventSink {
	ri := &redpanda.RedpandaInfo{}
	ri.SetBrokers()
	ri.SetTopics()

	return &WasmEventSink{es: es, encodingConfig: encodingConfig, ri: ri}
}

func (cs *WasmEventSink) InsertModuleEvents(req *abci.RequestFinalizeBlock, res *abci.ResponseFinalizeBlock) error {
	return nil
}

func (cs *WasmEventSink) EmitModuleEvents(req *abci.RequestFinalizeBlock, res *abci.ResponseFinalizeBlock) error {
	return nil
}

func (cs *WasmEventSink) ModuleName() string {
	return "wasm"
}

func (cs *WasmEventSink) EventSink() *psql.EventSink {
	return cs.es
}

func (cs *WasmEventSink) EncodingConfig() params.EncodingConfig {
	return cs.encodingConfig
}
