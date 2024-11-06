package tx

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/CosmWasm/wasmd/app/params"
	"github.com/CosmWasm/wasmd/indexer"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/state/indexer/sink/psql"
	"github.com/cometbft/cometbft/types"
	"github.com/hashicorp/go-hclog"
)

// EventSink is an indexer backend providing the tx/block index services.  This
// implementation stores records in a PostgreSQL database using the schema
// defined in state/indexer/sink/psql/schema.sql.
type TxEventSink struct {
	es             *psql.EventSink
	encodingConfig params.EncodingConfig
}

const (
	TableTxRequests = "tx_requests"
)

var _ indexer.ModuleEventSinkIndexer = (*TxEventSink)(nil)

func NewTxEventSinkIndexer(es *psql.EventSink, encodingConfig params.EncodingConfig) *TxEventSink {
	return &TxEventSink{es: es, encodingConfig: encodingConfig}
}

func (cs *TxEventSink) InsertModuleEvents(req *abci.RequestFinalizeBlock, res *abci.ResponseFinalizeBlock) error {
	// unmarshal txs
	hclog.Default().Debug("before unmarshal txs")
	for i, txBz := range req.Txs {
		cosmosTx, err := indexer.UnmarshalTxBz(cs, txBz)
		if err != nil {
			return err
		}
		fullMsgsBz, err := indexer.MarshalMsgsAny(cs.encodingConfig, cosmosTx.Body.Messages)
		if err != nil {
			return err
		}
		feeBz, err := json.Marshal(&cosmosTx.AuthInfo.Fee)
		if err != nil {
			return err
		}

		// Index the hash of the underlying transaction as a hex string.
		txHash := fmt.Sprintf("%X", types.Tx(txBz).Hash())
		if err := psql.RunInTransaction(cs.es.DB(), func(dbtx *sql.Tx) error {

			// just in case the cometbft indexer has not finished indexing block events, we index it by ourselves
			err := cs.es.IndexBlockEvents(types.EventDataNewBlockEvents{Height: req.Height, Events: res.Events, NumTxs: int64(len(req.Txs))})
			if err != nil {
				return err
			}

			// Find the block associated with this transaction. The block header
			// must have been indexed prior to the transactions belonging to it.
			blockID, err := psql.QueryWithID(dbtx, `
SELECT rowid FROM `+psql.TableBlocks+` WHERE height = $1 AND chain_id = $2;
`, req.Height, cs.es.ChainID())
			if err != nil {
				return err
			}

			// Insert a record for this tx_requests and capture its ID for indexing events.
			// NOTE: for tx index, it is the tx index in the list of txs. Ref: https://github.com/oraichain/cometbft/blob/5c0462aa0de4250a0c1ab43a80f8ea8adb84fa33/state/execution.go#L710; https://github.com/oraichain/cometbft/blob/5c0462aa0de4250a0c1ab43a80f8ea8adb84fa33/state/execution.go#L749
			_, err = psql.QueryWithID(dbtx, `
INSERT INTO `+TableTxRequests+` (block_id, index, height, created_at, tx_hash, messages, fee, memo)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT DO NOTHING
RETURNING rowid;
`, blockID, i, req.Height, req.Time, txHash, fullMsgsBz, string(feeBz), cosmosTx.Body.Memo)
			if err == sql.ErrNoRows {
				return nil // we already saw this transaction; quietly succeed
			} else if err != nil {
				return fmt.Errorf("indexing tx_requests: %w", err)
			}

			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

// FIXME: this is just for testing. Should have filters here based on events, height, chain id
func (cs *TxEventSink) SearchTxs(limit uint64, offset uint64) (uint64, error) {
	count := uint64(0)
	if err := psql.RunInTransaction(cs.es.DB(), func(dbtx *sql.Tx) error {

		// query txs. FIXME: Need filters and limit!
		row, err := dbtx.Query(`
SELECT
  tx_requests.index,
  json_attribute_events.chain_id,
  json_attribute_events.height,
  tx_requests.created_at,
  tx_requests.tx_hash,
  tx_requests.messages,
  tx_requests.memo,
  jsonb_agg(
    json_build_object('type', type, 'attributes', attributes)
  ) as events
FROM
  tx_requests
  JOIN json_attribute_events ON (json_attribute_events.block_id = tx_requests.block_id)
GROUP BY
  tx_requests.index,
  json_attribute_events.chain_id,
  json_attribute_events.height,
  tx_requests.created_at,
  tx_requests.tx_hash,
  tx_requests.messages,
  tx_requests.memo
ORDER BY
  json_attribute_events.height;
	`)
		if err != nil {
			return err
		}

		for {
			hasNext := row.Next()
			if !hasNext {
				break
			}
			count++
			var index int32
			var chainId string
			var height uint64
			var createdAt time.Time
			var txHash []byte
			var messages []byte
			var memo string
			var events string

			err = row.Scan(&index, &chainId, &height, &createdAt, &txHash, &messages, &memo, &events)
			if err != nil {
				panic(err)
			}
			msgsAny, err := indexer.UnmarshalMsgsBz(cs.encodingConfig, messages)
			if err != nil {
				return err
			}

			fmt.Println(index, chainId, height, createdAt, msgsAny, messages, memo, events)
		}
		return nil
	}); err != nil {
		return 0, err
	}
	return count, nil
}

func (cs *TxEventSink) EmitModuleEvents(req *abci.RequestFinalizeBlock, res *abci.ResponseFinalizeBlock) error {
	return nil
}

func (cs *TxEventSink) ModuleName() string {
	return "tx"
}

func (cs *TxEventSink) EventSink() *psql.EventSink {
	return cs.es
}

func (cs *TxEventSink) EncodingConfig() params.EncodingConfig {
	return cs.encodingConfig
}
