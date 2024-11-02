package indexer

import (
	"database/sql"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/state/indexer/sink/psql"
	"github.com/hashicorp/go-hclog"
)

// EventSink is an indexer backend providing the tx/block index services.  This
// implementation stores records in a PostgreSQL database using the schema
// defined in state/indexer/sink/psql/schema.sql.
type CustomEventSink struct {
	es *psql.EventSink
}

func IndexerFromConfig(conn, chainID string) (
	*CustomEventSink, error,
) {
	es, err := psql.NewEventSink(conn, chainID)
	if err != nil {
		return nil, fmt.Errorf("creating psql indexer: %w", err)
	}
	return &CustomEventSink{es: es}, nil
}

// runInTransaction executes query in a fresh database transaction.
// If query reports an error, the transaction is rolled back and the
// error from query is reported to the caller.
// Otherwise, the result of committing the transaction is returned.
func runInTransaction(db *sql.DB, query func(*sql.Tx) error) error {
	dbtx, err := db.Begin()
	if err != nil {
		return err
	}
	if err := query(dbtx); err != nil {
		_ = dbtx.Rollback() // report the initial error, not the rollback
		return err
	}
	return dbtx.Commit()
}

// FIXME: this is just testing logic for POC
func (cs *CustomEventSink) InsertModuleEvents(req abci.RequestFinalizeBlock, res abci.ResponseFinalizeBlock) error {
	var dest uint64
	err := runInTransaction(cs.es.DB(), func(dbtx *sql.Tx) error {
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
