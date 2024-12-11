package psql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/cosmos/gogoproto/proto"

	cfg "github.com/CosmWasm/wasmd/indexer/config"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/pubsub/query"
	"github.com/cometbft/cometbft/types"
)

// IndexBlockEvents indexes the specified block header, part of the
// EventSink interface.
func (es *EventSink) IndexBlockEvents(h types.EventDataNewBlockEvents) error {
	ts := time.Now().UTC()

	return RunInTransaction(es.store, func(dbtx *sql.Tx) error {
		// Add the block to the blocks table and report back its row ID for use
		// in indexing the events for the block.
		_, err := QueryWithID(dbtx,
			`
				INSERT INTO `+cfg.TableBlocks+` (height, chain_id, created_at)
				VALUES ($1, $2, $3)
				ON CONFLICT DO NOTHING
				RETURNING rowid;
			`,
			h.Height, es.chainID, ts)
		if err == sql.ErrNoRows {
			return nil // we already saw this block; quietly succeed
		} else if err != nil {
			return fmt.Errorf("indexing block header: %w", err)
		}

		// Insert the special block meta-event for height.
		// if err := insertEvents(dbtx, blockID, 0, []abci.Event{
		// 	makeIndexedEvent(types.BlockHeightKey, fmt.Sprint(h.Height)),
		// }); err != nil {
		// 	return fmt.Errorf("block meta-events: %w", err)
		// }
		// Insert all the block events. Order is important here,
		// We don't need block events -> reduce total number of attribute rows -> reduce query time
		// if err := insertEvents(dbtx, blockID, 0, h.Events); err != nil {
		// 	return fmt.Errorf("finalizeblock events: %w", err)
		// }
		return nil
	})
}

// IndexTxEvents indexes the specified txs, part of the
// EventSink interface.
func (es *EventSink) IndexTxEvents(txrs []*abci.TxResult) error {
	ts := time.Now().UTC()

	for _, txr := range txrs {
		// Encode the result message in protobuf wire format for indexing.
		resultData, err := proto.Marshal(txr)
		if err != nil {
			return fmt.Errorf("marshaling tx_result: %w", err)
		}

		// Index the hash of the underlying transaction as a hex string.
		txHash := fmt.Sprintf("%X", types.Tx(txr.Tx).Hash())

		var curBlockID uint32
		var curTxID uint32

		if err := RunInTransaction(es.store, func(dbtx *sql.Tx) error {
			// Find the block associated with this transaction. The block header
			// must have been indexed prior to the transactions belonging to it.
			blockID, err := QueryWithID(dbtx,
				`
					SELECT rowid FROM `+cfg.TableBlocks+` WHERE height = $1 AND chain_id = $2;
				`,
				txr.Height, es.chainID)
			if err != nil {
				return fmt.Errorf("finding block ID: %w", err)
			}
			curBlockID = blockID

			// Insert a record for this tx_result and capture its ID for indexing events.
			txID, err := QueryWithID(dbtx,
				`
					INSERT INTO `+cfg.TableTxResults+` (block_id, height, index, created_at, tx_hash, tx_result)
					VALUES ($1, $2, $3, $4, $5, $6)
					ON CONFLICT DO NOTHING
					RETURNING rowid;
				`,
				blockID, txr.Height, txr.Index, ts, txHash, resultData)
			if err == sql.ErrNoRows {
				return nil // we already saw this transaction; quietly succeed
			} else if err != nil {
				return fmt.Errorf("indexing tx_result: %w", err)
			}

			curTxID = txID

			return nil
		}); err != nil {
			return err
		}

		// Insert the special transaction meta-events for hash and height.
		if err := insertEvents(es.DB(), curBlockID, curTxID, []abci.Event{
			makeIndexedEvent(types.TxHashKey, txHash),
			makeIndexedEvent(types.TxHeightKey, fmt.Sprint(txr.Height)),
		}); err != nil {
			return fmt.Errorf("indexing transaction meta-events: %w", err)
		}
		// Index any events packaged with the transaction.
		if err := insertEvents(es.DB(), curBlockID, curTxID, txr.Result.Events); err != nil {
			return fmt.Errorf("indexing transaction events: %w", err)
		}
	}
	return nil
}

// SearchBlockEvents is not implemented by this sink, and reports an error for all queries.
func (es *EventSink) SearchBlockEvents(_ context.Context, _ *query.Query) ([]int64, error) {
	return nil, errors.New("block search is not supported via the postgres event sink")
}

// SearchTxEvents is not implemented by this sink, and reports an error for all queries.
func (es *EventSink) SearchTxEvents(_ context.Context, _ *query.Query) ([]*abci.TxResult, error) {
	return nil, errors.New("tx search is not supported via the postgres event sink")
}

// GetTxByHash is not implemented by this sink, and reports an error for all queries.
func (es *EventSink) GetTxByHash(_ []byte) (*abci.TxResult, error) {
	return nil, errors.New("getTxByHash is not supported via the postgres event sink")
}

// HasBlock is not implemented by this sink, and reports an error for all queries.
func (es *EventSink) HasBlock(_ int64) (bool, error) {
	return false, errors.New("hasBlock is not supported via the postgres event sink")
}

// Stop closes the underlying PostgreSQL database.
func (es *EventSink) Stop() error { return es.store.Close() }
