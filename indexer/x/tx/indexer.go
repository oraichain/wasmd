package tx

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/CosmWasm/wasmd/app/params"
	"github.com/CosmWasm/wasmd/indexer"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/state/indexer/sink/psql"
	"github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cosmostx "github.com/cosmos/cosmos-sdk/types/tx"
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
	for i, txBz := range req.Txs {
		// tx proto
		tx, err := cs.encodingConfig.TxConfig.TxDecoder()(txBz)
		if err != nil {
			hclog.Default().Debug("err decoder: ", err)
			tx, err = cs.encodingConfig.TxConfig.TxJSONDecoder()(txBz)
			if err != nil {
				panic(err)
			}
		}
		msgs := tx.GetMsgs()
		if err != nil {
			panic(err)
		}

		// try getting memo
		txWithMemo := tx.(sdk.TxWithMemo)
		memo := txWithMemo.GetMemo()

		// try getting fees
		feeTx := tx.(sdk.FeeTx)
		granter := feeTx.FeeGranter()
		payer := feeTx.FeePayer()
		fees := feeTx.GetFee()
		gas := feeTx.GetGas()
		fee := cosmostx.Fee{Amount: fees, GasLimit: gas, Payer: sdk.AccAddress(payer).String(), Granter: sdk.AccAddress(granter).String()}
		feeBz, err := json.Marshal(&fee)
		if err != nil {
			return err
		}

		msgsAny, err := cosmostx.SetMsgs(msgs)
		if err != nil {
			return err
		}
		msgsBz := [][]byte{}
		for _, msg := range msgsAny {
			msgMarshal, err := cs.encodingConfig.Codec.Marshal(msg)
			if err != nil {
				return err
			}
			msgsBz = append(msgsBz, msgMarshal)
		}

		fullMsgsBz, err := cs.encodingConfig.Amino.Marshal(msgsBz)
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
INSERT INTO `+TableTxRequests+` (block_id, index, created_at, tx_hash, messages, fee, memo)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT DO NOTHING
RETURNING rowid;
`, blockID, i, req.Time, txHash, fullMsgsBz, string(feeBz), memo)
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

func (cs *TxEventSink) EmitModuleEvents(req *abci.RequestFinalizeBlock, res *abci.ResponseFinalizeBlock) error {
	panic("Not implemented")
}

func (cs *TxEventSink) ModuleName() string {
	return "tx"
}
