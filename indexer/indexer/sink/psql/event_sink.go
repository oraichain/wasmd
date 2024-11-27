package psql

import (
	"database/sql"

	cfg "github.com/CosmWasm/wasmd/indexer/indexer/config"
)

// EventSink is an indexer backend providing the tx/block index services.  This
// implementation stores records in a PostgreSQL database using the schema
// defined in state/indexer/sink/psql/schema.sql.
type EventSink struct {
	store   *sql.DB
	chainID string
}

// NewEventSink constructs an event sink associated with the PostgreSQL
// database specified by connStr. Events written to the sink are attributed to
// the specified chainID.
func NewEventSink(connStr, chainID string) (*EventSink, error) {
	db, err := sql.Open(cfg.DrviverName, connStr)
	if err != nil {
		return nil, err
	}

	return &EventSink{
		store:   db,
		chainID: chainID,
	}, nil
}

// NewEventSinkFromDB constructs an event sink associated with the PostgreSQL from db input
func NewEventSinkFromDB(db *sql.DB, chainID string) *EventSink {
	return &EventSink{
		store:   db,
		chainID: chainID,
	}
}

// DB returns the underlying Postgres connection used by the sink.
// This is exported to support testing.
func (es *EventSink) DB() *sql.DB { return es.store }

func (es *EventSink) ChainID() string {
	return es.chainID
}
