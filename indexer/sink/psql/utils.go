package psql

import (
	"database/sql"
	"fmt"
	"strings"
	"unicode"

	cfg "github.com/CosmWasm/wasmd/indexer/config"
	abci "github.com/cometbft/cometbft/abci/types"
)

// RunInTransaction executes query in a fresh database transaction.
// If query reports an error, the transaction is rolled back and the
// error from query is reported to the caller.
// Otherwise, the result of committing the transaction is returned.
func RunInTransaction(db *sql.DB, query func(*sql.Tx) error) error {
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

// QueryWithID executes the specified SQL query with the given arguments,
// expecting a single-row, single-column result containing an ID. If the query
// succeeds, the ID from the result is returned.
func QueryWithID(tx *sql.Tx, query string, args ...interface{}) (uint32, error) {
	var id uint32
	if err := tx.QueryRow(query, args...).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

// QueryWithID executes the specified SQL query with the given arguments,
// expecting a single-row, single-column result containing an ID. If the query
// succeeds, the ID from the result is returned.
func QueryWithIDNoTx(tx *sql.DB, query string, args ...interface{}) (uint32, error) {
	var id uint32
	if err := tx.QueryRow(query, args...).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

// insertEvents inserts a slice of events and any indexed attributes of those
// events into the database associated with dbtx.
//
// If txID > 0, the event is attributed to the transaction with that
// ID; otherwise it is recorded as a block event.
func insertEvents(dbtx *sql.DB, blockID, txID uint32, evts []abci.Event) error {
	// Populate the transaction ID field iff one is defined (> 0).
	var txIDArg interface{}
	if txID > 0 {
		txIDArg = txID
	}

	// Index any events packaged with the transaction.
	const (
		insertEventQuery = `
			INSERT INTO ` + cfg.TableEvents + ` (block_id, tx_id, type)
			VALUES ($1, $2, $3)
			RETURNING rowid;
		`
		insertAttributeQuery = `
			INSERT INTO ` + cfg.TableAttributes + ` (event_id, key, composite_key, value)
			VALUES ($1, $2, $3, $4);
		`
	)

	// Add each event to the events table, and retrieve its row ID to use when
	// adding any attributes the event provides.
	for _, evt := range evts {
		// Skip events with an empty type.
		if evt.Type == "" {
			continue
		}

		eid, err := QueryWithIDNoTx(dbtx, insertEventQuery, blockID, txIDArg, evt.Type)
		if err != nil {
			return fmt.Errorf(fmt.Sprintf("Error inserting event query: %v of event %v: %v: %v\n", blockID, txIDArg, evt.Type, err), "")
		}

		// Add any attributes flagged for indexing.
		for _, attr := range evt.Attributes {
			if !attr.Index {
				continue
			}
			compositeKey := evt.Type + "." + attr.Key
			// ignore block_bloom, it is causing some errors when inserting
			if compositeKey == "block_bloom.bloom" {
				continue
			}
			// max length of a row in psql
			if len(attr.Value) > 8191 {
				continue
			}
			attrValue := attr.Value
			if hasNonPrintableChars(attr.Value) {
				// convert to hex to safely store the value
				attrValue = fmt.Sprintf("%x\n", []byte(attr.Value))
			}
			if _, err := dbtx.Exec(insertAttributeQuery, eid, attr.Key, compositeKey, attrValue); err != nil {
				// since we can't control the values of the attrs, it's best we ignore attrs that have errors
				fmt.Printf("error processing attr: %v of event %v: %v\n", attr, evt.Type, err)
				continue
			}
		}
	}
	return nil
}

// makeIndexedEvent constructs an event from the specified composite key and
// value. If the key has the form "type.name", the event will have a single
// attribute with that name and the value; otherwise the event will have only
// a type and no attributes.
func makeIndexedEvent(compositeKey, value string) abci.Event {
	i := strings.Index(compositeKey, ".")
	if i < 0 {
		return abci.Event{Type: compositeKey}
	}
	return abci.Event{Type: compositeKey[:i], Attributes: []abci.EventAttribute{
		{Key: compositeKey[i+1:], Value: value, Index: true},
	}}
}

func MakeIndexedEvent(compositeKey, value string) abci.Event {
	return makeIndexedEvent(compositeKey, value)
}

func hasNonPrintableChars(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) {
			return true // Non-printable character found
		}
	}
	return false // All characters are printable
}
