package tx

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/CosmWasm/wasmd/app/params"
	"github.com/CosmWasm/wasmd/indexer"
	redpanda "github.com/CosmWasm/wasmd/streaming/redpanda"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtquery "github.com/cometbft/cometbft/libs/pubsub/query"
	"github.com/cometbft/cometbft/libs/pubsub/query/syntax"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	cometbftindexer "github.com/cometbft/cometbft/state/indexer"
	"github.com/cometbft/cometbft/state/indexer/sink/psql"
	"github.com/cometbft/cometbft/state/txindex/kv"
	cmttypes "github.com/cometbft/cometbft/types"
)

// EventSink is an indexer backend providing the tx/block index services.  This
// implementation stores records in a PostgreSQL database using the schema
// defined in state/indexer/sink/psql/schema.sql.
type TxEventSink struct {
	es             *psql.EventSink
	encodingConfig params.EncodingConfig
	ri             *redpanda.RedpandaInfo
}

const (
	TxSearchLimit = uint32(100000)
)

var _ indexer.ModuleEventSinkIndexer = (*TxEventSink)(nil)

func NewTxEventSinkIndexer(es *psql.EventSink, encodingConfig params.EncodingConfig) *TxEventSink {
	ri := &redpanda.RedpandaInfo{}
	ri.SetBrokers()
	ri.SetTopics()

	return &TxEventSink{es: es, encodingConfig: encodingConfig, ri: ri}
}

func (cs *TxEventSink) InsertModuleEvents(req *abci.RequestFinalizeBlock, res *abci.ResponseFinalizeBlock) error {
	return nil
}

func (cs *TxEventSink) TxSearch(_ *rpctypes.Context, query string, _limit *int, txHash string) (*ctypes.ResultTxSearch, error) {

	// if tx hash is not empty -> we query via tx hash directly and ignore tx search txs
	if txHash != "" {
		txResponses, err := cs.GetTxByHash(txHash)
		if err != nil {
			return nil, err
		}
		return &ctypes.ResultTxSearch{Txs: txResponses, TotalCount: 1}, nil
	}

	if query == "" {
		latestBlock, err := cs.GetLatestBlock()
		if err != nil {
			return nil, err
		}
		// sneak peak 10 latest blocks if leave empty
		query = fmt.Sprintf("tx.height >= %d AND tx.height <= %d", max(0, latestBlock-10), latestBlock)
	}
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

// TODO: handle filter condition -> done
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
	_, _, heightRange := cometbftindexer.LookForRangesWithHeight(conditions)
	heightInfo.SetheightRange(heightRange)
	whereConditions, args, argsCount := CreateHeightRangeWhereConditions(heightInfo)
	whereConditions, err := cs.createCursorPaginationCondition(whereConditions)
	if err != nil {
		return nil, 0, err
	}
	filterTableClause, filterArgs, err := CreateNonHeightConditionFilterTable(conditions, argsCount)
	if err != nil {
		return nil, 0, err
	}
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
    tx_id,
		attributes.composite_key,
		attributes.value
  FROM
    events
		JOIN filtered_heights fh on (fh.rowid = events.tx_id)
    JOIN attributes ON (events.rowid = attributes.event_id)
  WHERE tx_id is NOT null 
	ORDER BY tx_id DESC 
	),
	-- filter txs based on input composite key conditions
	%s
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
	admin := cs.ri.GetAdmin()
	if admin == nil {
		cs.ri.SetAdmin()
		admin = cs.ri.GetAdmin()
	}

	producer := cs.ri.GetProducer()
	if producer == nil {
		cs.ri.SetProducer()
		producer = cs.ri.GetProducer()
	}

	for i, tx := range req.Txs {
		cosmosTx, err := indexer.UnmarshalTxBz(cs, tx)
		if err != nil {
			return err
		}

		// get topic for tx
		var topics []string
		for _, message := range cosmosTx.Body.Messages {
			typeUrl := strings.Split(message.TypeUrl, "/")[1]
			module := strings.Split(typeUrl, ".")[1]
			topic := "REDPANDA_TOPIC_" + strings.ToUpper(module)

			if !admin.IsTopicExist(topic) {
				err := admin.CreateTopic(topic)
				if err != nil {
					return err
				}
			}

			topics = append(topics, topic)
		}

		txHashBz := cmttypes.Tx(tx).Hash()
		topicMsg := ctypes.ResultTx{Height: req.Height, Hash: txHashBz, TxResult: *res.TxResults[i], Index: uint32(i), Tx: tx, Timestamp: req.Time.Format(time.RFC3339)}

		err = producer.SendToRedpanda(topics, topicMsg)
		if err != nil {
			return err
		}
	}

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

func CreateHeightRangeWhereConditions(heightInfo kv.HeightInfo) (whereConditions string, vals []interface{}, argsCount *int) {
	// args count is used to increment parameterized arguments
	initialCount := 1
	argsCount = &initialCount
	// prioritize range conditions
	if isHeightRangeNotEmpty(heightInfo.HeightRange()) {
		value := heightInfo.HeightRange()
		ops, values := detectQueryRangeBound(value)
		whereConditions += "WHERE"
		for i, operator := range ops {
			if i == len(ops)-1 {
				whereConditions += fmt.Sprintf(" height %s $%d", operator, *argsCount)
			} else {
				whereConditions += fmt.Sprintf(" height %s $%d AND", operator, *argsCount)
			}

			*argsCount++
		}
		vals = values
		return whereConditions, vals, argsCount
	}
	// if there's no range, and has eq condition -> handle it
	if heightInfo.Height() != 0 {
		return fmt.Sprintf("WHERE height = $%d", *argsCount), []interface{}{heightInfo.Height()}, argsCount
	}
	return "", nil, &initialCount
}

func isHeightRangeNotEmpty(heightRange cometbftindexer.QueryRange) bool {
	return heightRange.LowerBound != nil || heightRange.UpperBound != nil
}

func (cs *TxEventSink) createCursorPaginationCondition(whereCondition string) (string, error) {
	if whereCondition != "" {
		return whereCondition, nil
	}
	// if the whereCondition is empty -> we create the pagination cursor based on the latest height
	height, err := cs.GetLatestBlock()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("WHERE height <= %d", height), nil
}

func (cs *TxEventSink) GetLatestBlock() (int64, error) {
	var height int64
	if err := psql.RunInTransaction(cs.es.DB(), func(dbtx *sql.Tx) error {
		// Find the block associated with this transaction. The block header
		// must have been indexed prior to the transactions belonging to it.
		if err := dbtx.QueryRow(`
SELECT height FROM ` + psql.TableBlocks + ` order by height desc limit 1;
`).Scan(&height); err != nil {
			return fmt.Errorf("error finding latest block: %w", err)
		}
		return nil
	}); err != nil {
		return 0, err
	}
	return height, nil
}

func (cs *TxEventSink) GetTxByHash(txHash string) ([]*ctypes.ResultTx, error) {
	var height int64
	var createdAt time.Time
	var txResultBz []byte
	var txResult abci.TxResult
	txResponses := []*ctypes.ResultTx{}

	if err := psql.RunInTransaction(cs.es.DB(), func(dbtx *sql.Tx) error {
		// Find the block associated with this transaction. The block header
		// must have been indexed prior to the transactions belonging to it.
		if err := dbtx.QueryRow(fmt.Sprintf(`
	SELECT 
		tx_results.height,
		tx_results.created_at,
		tx_results.tx_result
 	FROM %s
	WHERE tx_results.tx_hash = '%s'`, psql.TableTxResults, txHash)).Scan(&height, &createdAt, &txResultBz); err != nil {
			return fmt.Errorf("error finding tx by hash: %w with hash: %s", err, txHash)
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

		return nil
	}); err != nil {
		return nil, err
	}
	return txResponses, nil
}

func CreateNonHeightConditionFilterTable(conditions []syntax.Condition, argsCount *int) (filterTableClause string, vals []interface{}, err error) {
	tableName := "ftea"
	filterTableClause += "filtered_tx_ids as ("
	filterTxs := func(tableName string) string {
		return fmt.Sprintf("\nselect distinct tx_id \nfrom filtered_tx_event_attributes %s \n", tableName)
	}
	tableNameDelta := int8(1)
	hasNonheightCondition := false
	for i, condition := range conditions {
		// ignore since we already covered tx.height elsewhere
		if condition.Tag == cmttypes.TxHeightKey {
			continue
		}
		completeTableName := fmt.Sprintf("%s%d", tableName, tableNameDelta)
		tableNameDelta++
		hasNonheightCondition = true
		whereClause := fmt.Sprintf("%sWHERE %s.composite_key = $%d \n", filterTxs(completeTableName), completeTableName, *argsCount)
		*argsCount++
		vals = append(vals, condition.Tag)
		whereValueClause, val, err := matchNonHeightCondition(condition, completeTableName, argsCount)
		if err != nil {
			return "", vals, err
		}
		whereClause += whereValueClause
		vals = append(vals, val)
		filterTableClause += whereClause

		// if it's not the last condition -> add INTERSECT keyword to intersect the tables for AND condition
		// TODO: If we allow OR keyword -> switch case to UNION
		if i < len(conditions)-1 {
			filterTableClause += "INTERSECT"
		}
	}
	// empty table clause, meaning that there are no other clauses -> return filtered_tx_ids bare minimum
	if !hasNonheightCondition {
		return fmt.Sprintf("%s\n%s)", filterTableClause, filterTxs(tableName)), vals, nil
	}
	return fmt.Sprintf("%s)\n", filterTableClause), vals, nil
}

func matchNonHeightCondition(condition syntax.Condition, completeTableName string, argsCount *int) (whereClause string, val interface{}, err error) {
	opStr, err := convertOpToOpStr(condition.Op)
	if err != nil {
		return "", nil, err
	}
	clause := fmt.Sprintf("AND %s.value %s $%d \n", completeTableName, opStr, *argsCount)
	*argsCount++
	return clause, conditionArg(condition), nil
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

func convertOpToOpStr(op syntax.Token) (string, error) {
	switch op {
	case syntax.TEq:
		return "=", nil
	case syntax.TGeq:
		return ">=", nil
	case syntax.TLeq:
		return "<=", nil
	case syntax.TLt:
		return "<", nil
	case syntax.TGt:
		return ">", nil
	default:
		return "", fmt.Errorf("error converting op to op str. The op doesn't match any defined op")
	}
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
