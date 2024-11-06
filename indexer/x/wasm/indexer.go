package wasm

import (
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

func (cs *WasmEventSink) insertTxEvents(req *abci.RequestFinalizeBlock, res *abci.ResponseFinalizeBlock) ([]*abci.TxResult, error) {
	txResults := []*abci.TxResult{}
	for i, tx := range req.Txs {
		txResult := &abci.TxResult{
			Height: req.Height,
			Index:  uint32(i),
			Tx:     tx,
			Result: *res.TxResults[i],
			Time:   &req.Time,
		}
		txResults = append(txResults, txResult)
	}
	// we need tx events to get wasm txs. If the cometbft indexer already inserts it -> nothing will happen
	err := cs.es.IndexTxEvents(txResults)
	if err != nil {
		return nil, err
	}
	return txResults, nil
}

func (cs *WasmEventSink) InsertModuleEvents(req *abci.RequestFinalizeBlock, res *abci.ResponseFinalizeBlock) error {
	_, err := cs.insertTxEvents(req, res)
	if err != nil {
		return err
	}
	return nil
}

func (cs *WasmEventSink) EmitModuleEvents(req *abci.RequestFinalizeBlock, res *abci.ResponseFinalizeBlock) error {
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
