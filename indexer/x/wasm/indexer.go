package wasm

import (
	"database/sql"
	"fmt"

	"github.com/CosmWasm/wasmd/indexer"
	"github.com/CosmWasm/wasmd/x/wasm/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/state/indexer/sink/psql"
	"github.com/hashicorp/go-hclog"
)

// EventSink is an indexer backend providing the tx/block index services.  This
// implementation stores records in a PostgreSQL database using the schema
// defined in state/indexer/sink/psql/schema.sql.
type WasmEventSink struct {
	es *psql.EventSink
}

var _ indexer.ModuleEventSinkIndexer = (*WasmEventSink)(nil)

func NewWasmEventSinkIndexer(es *psql.EventSink) *WasmEventSink {
	return &WasmEventSink{es: es}
}

// FIXME: this is just testing logic for POC
func (cs *WasmEventSink) InsertModuleEvents(req *abci.RequestFinalizeBlock, res *abci.ResponseFinalizeBlock) error {
	var dest uint64
	err := indexer.RunInTransaction(cs.es.DB(), func(dbtx *sql.Tx) error {
		res, err := dbtx.Query("select height from blocks limit 1")
		if err != nil {
			return err
		}
		res.Next()
		err = res.Scan(&dest)
		if err != nil {
			return err
		}

		return nil
	})
	hclog.Default().Error(fmt.Sprintln("wasm res: ", dest))
	return err
}

func (cs *WasmEventSink) EmitModuleEvents(req *abci.RequestFinalizeBlock, res *abci.ResponseFinalizeBlock) error {
	panic("Not implemented")
}

func (cs *WasmEventSink) ModuleName() string {
	return types.ModuleName
}
