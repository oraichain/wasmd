package tx

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/CosmWasm/wasmd/app/params"
	"github.com/CosmWasm/wasmd/indexer"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/pubsub/query"
	"github.com/cometbft/cometbft/libs/pubsub/query/syntax"
	cometbftindexer "github.com/cometbft/cometbft/state/indexer"
	"github.com/cometbft/cometbft/state/indexer/sink/psql"
	"github.com/cometbft/cometbft/state/txindex/kv"
	"github.com/cometbft/cometbft/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
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
		feeBz, err := cs.encodingConfig.Codec.MarshalJSON(cosmosTx.AuthInfo.Fee)
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

// TODO: add limit & filters based on non-height conditions
func (cs *TxEventSink) SearchTxs(q *query.Query) ([]*txtypes.GetTxResponse, uint64, error) {
	count := uint64(0)
	txResponses := []*txtypes.GetTxResponse{}

	conditions := q.Syntax()
	// conditions to skip because they're handled before "everything else"
	// If we are not matching events and tx.height = 3 occurs more than once, the later value will
	// overwrite the first one.
	conditions, heightInfo := kv.DedupHeight(conditions)

	// extract ranges
	// if both upper and lower bounds exist, it's better to get them in order not
	// no iterate over kvs that are not within range.
	ranges, _ := lookForRangesWithHeight(conditions)
	_, ok := ranges[types.TxHeightKey]
	heightInfo.SetheightRange(ranges[types.TxHeightKey])
	whereConditions, args, _ := CreateHeightRangeWhereConditions(heightInfo, ok)
	queryClause := `
	select
  tx_requests.height,
  tx_requests.created_at,
  tx_requests.tx_hash,
  messages,
  memo,
  fee,
	code, 
	logs, 
	info, 
	gas_wanted, 
	gas_used, 
	codespace,
  jsonb_agg(
    distinct jsonb_build_object('type', type, 'attributes', attributes)
  ) as events
from tx_requests
join tx_results txr on (tx_requests.block_id = txr.block_id)
join json_attribute_events jae on (tx_requests.height = jae.height) ` + whereConditions +
		` group by
  tx_requests.height,
  tx_requests.created_at,
  tx_requests.tx_hash,
  messages,
  memo,
  fee,
	code, 
	logs, 
	info, 
	gas_wanted, 
	gas_used, 
	codespace;
	`
	if err := psql.RunInTransaction(cs.es.DB(), func(dbtx *sql.Tx) error {

		// query txs. FIXME: Need filters and limit!
		row, err := dbtx.Query(queryClause, args...)
		if err != nil {
			return err
		}

		for {
			hasNext := row.Next()
			if !hasNext {
				break
			}
			count++
			var height int64
			var createdAt time.Time
			var txHash string
			var messages []byte
			var memo string
			var fee string
			var code uint32
			var logs string
			var info string
			var gasWanted int64
			var gasUsed int64
			var codespace string
			var events string

			err = row.Scan(&height, &createdAt, &txHash, &messages, &memo, &fee, &code, &logs, &info, &gasWanted, &gasUsed, &codespace, &events)
			if err != nil {
				panic(err)
			}
			msgsAny, err := indexer.UnmarshalMsgsBz(cs.encodingConfig, messages)
			if err != nil {
				return err
			}

			var feeProto txtypes.Fee
			if err := cs.encodingConfig.Codec.UnmarshalJSON([]byte(fee), &feeProto); err != nil {
				return err
			}

			var eventsProto []abci.Event
			if err := cs.encodingConfig.Amino.UnmarshalJSON([]byte(events), &eventsProto); err != nil {
				return err
			}

			txBody := txtypes.TxBody{
				Messages: msgsAny,
				Memo:     memo,
			}
			cosmosTx := txtypes.Tx{Body: &txBody, AuthInfo: &txtypes.AuthInfo{Fee: &feeProto}}
			txResponse := cosmostypes.TxResponse{Height: height, TxHash: txHash, Codespace: codespace, Code: code, Info: info, RawLog: logs, GasWanted: gasWanted, GasUsed: gasUsed, Events: eventsProto}
			txResponses = append(txResponses, &txtypes.GetTxResponse{Tx: &cosmosTx, TxResponse: &txResponse})
		}
		return nil
	}); err != nil {
		return nil, 0, err
	}
	return txResponses, count, nil
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

func CreateHeightRangeWhereConditions(heightInfo kv.HeightInfo, hasHeightRange bool) (whereConditions string, vals []interface{}, argsCount int) {
	// args count is used to increment parameterized arguments
	argsCount = 1
	// prioritize range conditions
	if hasHeightRange {
		value := heightInfo.HeightRange()
		ops, values := detectQueryRangeBound(value)
		whereConditions += "WHERE"
		for i, operator := range ops {
			if i == len(ops)-1 {
				whereConditions += fmt.Sprintf(" %s.height %s $%d", TableTxRequests, operator, argsCount)
			} else {
				whereConditions += fmt.Sprintf(" %s.height %s $%d AND", TableTxRequests, operator, argsCount)
			}

			argsCount++
		}
		vals = values
		return whereConditions, vals, argsCount
	}
	// if there's no range, and has eq condition -> handle it
	if heightInfo.Height() != 0 {
		return fmt.Sprintf("WHERE %s.height = $%d", TableTxRequests, argsCount), []interface{}{heightInfo.Height()}, argsCount
	}
	return "", nil, 0
}

func detectQueryRangeBound(value cometbftindexer.QueryRange) (ops []string, vals []interface{}) {
	if value.LowerBound != nil {
		operator := ">"
		if value.IncludeLowerBound {
			operator = ">="
		}
		ops = append(ops, operator)
		vals = append(vals, value.LowerBound.(int64))
	}
	if value.UpperBound != nil {
		operator := "<"
		if value.IncludeUpperBound {
			operator = "<="
		}
		upper := value.UpperBound.(int64)
		ops = append(ops, operator)
		vals = append(vals, upper)
	}
	return ops, vals
}

// lookForRangesWithHeight returns a mapping of QueryRanges and the matching indexes in
// the provided query conditions.
func lookForRangesWithHeight(conditions []syntax.Condition) (queryRange cometbftindexer.QueryRanges, nonRangeIndexes []int) {
	queryRange = make(cometbftindexer.QueryRanges)
	for i, c := range conditions {
		if cometbftindexer.IsRangeOperation(c.Op) {
			r, ok := queryRange[c.Tag]
			if !ok {
				r = cometbftindexer.QueryRange{Key: c.Tag}
			}

			switch c.Op {
			case syntax.TGt:
				r.LowerBound = conditionArg(c)

			case syntax.TGeq:
				r.IncludeLowerBound = true
				r.LowerBound = conditionArg(c)

			case syntax.TLt:
				r.UpperBound = conditionArg(c)

			case syntax.TLeq:
				r.IncludeUpperBound = true
				r.UpperBound = conditionArg(c)
			}

			queryRange[c.Tag] = r
		} else {
			nonRangeIndexes = append(nonRangeIndexes, i)
		}
	}

	return queryRange, nonRangeIndexes
}

func conditionArg(c syntax.Condition) interface{} {
	if c.Arg == nil {
		return nil
	}
	switch c.Arg.Type {
	case syntax.TNumber:
		num, _ := c.Arg.Number().Int64()
		return num
	case syntax.TTime, syntax.TDate:
		return c.Arg.Time()
	default:
		return c.Arg.Value() // string
	}
}
