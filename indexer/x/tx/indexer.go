package tx

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/CosmWasm/wasmd/app/params"
	"github.com/CosmWasm/wasmd/indexer"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtquery "github.com/cometbft/cometbft/libs/pubsub/query"
	"github.com/cometbft/cometbft/libs/pubsub/query/syntax"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	cometbftindexer "github.com/cometbft/cometbft/state/indexer"
	"github.com/cometbft/cometbft/state/indexer/sink/psql"
	"github.com/cometbft/cometbft/state/txindex/kv"
)

// EventSink is an indexer backend providing the tx/block index services.  This
// implementation stores records in a PostgreSQL database using the schema
// defined in state/indexer/sink/psql/schema.sql.
type TxEventSink struct {
	es             *psql.EventSink
	encodingConfig params.EncodingConfig
}

const (
	TxSearchLimit = uint32(100000)
)

var _ indexer.ModuleEventSinkIndexer = (*TxEventSink)(nil)

func NewTxEventSinkIndexer(es *psql.EventSink, encodingConfig params.EncodingConfig) *TxEventSink {
	return &TxEventSink{es: es, encodingConfig: encodingConfig}
}

func (cs *TxEventSink) InsertModuleEvents(req *abci.RequestFinalizeBlock, res *abci.ResponseFinalizeBlock) error {
	return nil
}

// TODO: handle empty query, handle filter condition, handle query tx hash -> done
func (cs *TxEventSink) TxSearch(_ *rpctypes.Context, query string, _limit *int) (*ctypes.ResultTxSearch, error) {
	q, err := cmtquery.New(query)
	if err != nil {
		return nil, err
	}
	limit := TxSearchLimit
	if _limit != nil {
		limit = uint32(*_limit)
	}
	txResponses, count, err := cs.SearchTxs(q, limit)
	if err != nil {
		return nil, err
	}
	return &ctypes.ResultTxSearch{Txs: txResponses, TotalCount: int(count)}, nil
}

// TODO: add limit & filters based on non-height conditions
func (cs *TxEventSink) SearchTxs(q *cmtquery.Query, limit uint32) ([]*ctypes.ResultTx, uint64, error) {
	count := uint64(0)
	txResponses := []*ctypes.ResultTx{}

	conditions := q.Syntax()
	// conditions to skip because they're handled before "everything else"
	// If we are not matching events and tx.height = 3 occurs more than once, the later value will
	// overwrite the first one.
	conditions, heightInfo := kv.DedupHeight(conditions)

	// extract ranges
	// if both upper and lower bounds exist, it's better to get them in order not
	// no iterate over kvs that are not within range.
	ranges, indexes, heightRange := cometbftindexer.LookForRangesWithHeight(conditions)
	heightInfo.SetheightRange(heightRange)
	whereConditions, args, argsCount := CreateHeightRangeWhereConditions(heightInfo)
	whereConditions, err := cs.createCursorPaginationCondition(whereConditions)
	if err != nil {
		return nil, 0, err
	}
	filterTableClause, filterArgs := CreateNonHeightConditionFilterTable(conditions, ranges, indexes, argsCount)
	queryClause := fmt.Sprintf(`
	-- get all heights <= x that have txs, and limit the number of heights to y
	WITH filtered_heights AS (
    SELECT distinct tr.rowid, height
    FROM tx_results tr %s
    ORDER BY height desc
		LIMIT %d
	),
	-- filter all attributes within the filtered heights. This makes sure we still have limit & pagination without filtering out events
	filtered_tx_event_attributes as (
  SELECT
    events.block_id,
    height,
    tx_id
  FROM
    events
		JOIN filtered_heights fh on (fh.rowid = events.tx_id)
    JOIN attributes ON (events.rowid = attributes.event_id)
  WHERE tx_id is NOT null 
	ORDER BY tx_id DESC 
	),
	-- filter txs based on input composite key conditions
	filtered_tx_ids as (
		select distinct tx_id
		from filtered_tx_event_attributes te
		%s
	)
	-- join everything to get the final table with sufficient data
	select
		tr.height,
		tr.created_at,
		tr.tx_hash,
		tr.tx_result
	from
		filtered_tx_ids ftx
		join tx_results tr on tr.rowid = ftx.tx_id
	ORDER BY ftx.tx_id DESC;
	`, whereConditions, min(TxSearchLimit, limit), filterTableClause)
	if err := psql.RunInTransaction(cs.es.DB(), func(dbtx *sql.Tx) error {

		// query txs. FIXME: Need filters and limit!
		row, err := dbtx.Query(queryClause, append(args, filterArgs...)...)
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
			var txResultBz []byte
			var txResult abci.TxResult

			err = row.Scan(&height, &createdAt, &txHash, &txResultBz)
			if err != nil {
				return err
			}

			if err := cs.encodingConfig.Codec.Unmarshal(txResultBz, &txResult); err != nil {
				return err
			}
			txHashBz, err := hex.DecodeString(txHash)
			if err != nil {
				return err
			}

			txResponse := ctypes.ResultTx{Height: height, Hash: txHashBz, TxResult: txResult.Result, Index: txResult.Index, Tx: txResult.Tx}
			if txResult.Time != nil {
				txResponse.Timestamp = txResult.Time.Format(time.RFC3339)
			}
			txResponses = append(txResponses, &txResponse)
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

func CreateHeightRangeWhereConditions(heightInfo kv.HeightInfo) (whereConditions string, vals []interface{}, argsCount int) {
	// args count is used to increment parameterized arguments
	argsCount = 1
	// prioritize range conditions
	if isHeightRangeNotEmpty(heightInfo.HeightRange()) {
		value := heightInfo.HeightRange()
		ops, values := detectQueryRangeBound(value)
		whereConditions += "WHERE"
		for i, operator := range ops {
			if i == len(ops)-1 {
				whereConditions += fmt.Sprintf(" height %s $%d", operator, argsCount)
			} else {
				whereConditions += fmt.Sprintf(" height %s $%d AND", operator, argsCount)
			}

			argsCount++
		}
		vals = values
		return whereConditions, vals, argsCount
	}
	// if there's no range, and has eq condition -> handle it
	if heightInfo.Height() != 0 {
		return fmt.Sprintf("WHERE height = $%d", argsCount), []interface{}{heightInfo.Height()}, argsCount
	}
	return "", nil, 0
}

func isHeightRangeNotEmpty(heightRange cometbftindexer.QueryRange) bool {
	return heightRange.LowerBound != nil || heightRange.UpperBound != nil
}

func (cs *TxEventSink) createCursorPaginationCondition(whereCondition string) (string, error) {
	if whereCondition != "" {
		return whereCondition, nil
	}
	// if the whereCondition is empty -> we create the pagination cursor based on the latest height
	var height int64
	if err := psql.RunInTransaction(cs.es.DB(), func(dbtx *sql.Tx) error {
		// Find the block associated with this transaction. The block header
		// must have been indexed prior to the transactions belonging to it.
		if err := dbtx.QueryRow(`
SELECT height FROM ` + psql.TableBlocks + ` order by height desc limit 1;
`).Scan(&height); err != nil {
			return fmt.Errorf("finding block height: %w", err)
		}
		return nil
	}); err != nil {
		return "", err
	}
	return fmt.Sprintf("WHERE height <= %d", height), nil
}

func CreateNonHeightConditionFilterTable(conditions []syntax.Condition, ranges cometbftindexer.QueryRanges, rangeIndexes []int, argsCount int) (filterTableClause string, vals []interface{}) {
	// TODO: add filter conditions to handle non-height filters
	return "", []interface{}{}
}

func detectQueryRangeBound(value cometbftindexer.QueryRange) (ops []string, vals []interface{}) {
	if value.LowerBound != nil {
		operator := ">"
		if value.IncludeLowerBound {
			operator = ">="
		}
		ops = append(ops, operator)
		val, _ := value.LowerBound.(*big.Float).Int64()
		vals = append(vals, val)
	}
	if value.UpperBound != nil {
		operator := "<"
		if value.IncludeUpperBound {
			operator = "<="
		}
		ops = append(ops, operator)
		upper, _ := value.UpperBound.(*big.Float).Int64()
		vals = append(vals, upper)
	}
	return ops, vals
}
