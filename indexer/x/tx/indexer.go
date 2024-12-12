package tx

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/CosmWasm/wasmd/app/params"
	indexerCfg "github.com/CosmWasm/wasmd/indexer/config"
	"github.com/CosmWasm/wasmd/indexer/sink/psql"
	indexerType "github.com/CosmWasm/wasmd/indexer/types"
	indexerUtil "github.com/CosmWasm/wasmd/indexer/utils"
	redpanda "github.com/CosmWasm/wasmd/streaming/redpanda"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtquery "github.com/cometbft/cometbft/libs/pubsub/query"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	cometbftindexer "github.com/cometbft/cometbft/state/indexer"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/hashicorp/go-hclog"
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

var _ indexerType.ModuleEventSinkIndexer = (*TxEventSink)(nil)

func NewTxEventSinkIndexer(
	es *psql.EventSink,
	encodingConfig params.EncodingConfig,
	ri *redpanda.RedpandaInfo,
) *TxEventSink {
	return &TxEventSink{es: es, encodingConfig: encodingConfig, ri: ri}
}

func (cs *TxEventSink) InsertModuleEvents(req *abci.RequestFinalizeBlock, res *abci.ResponseFinalizeBlock) error {
	height := req.GetHeight()
	numTxs := int64(len(req.GetTxs()))

	txResults := []*abci.TxResult{}
	for i := int64(0); i < numTxs; i++ {
		txResult := abci.TxResult{
			Height: height,
			Index:  uint32(i),
			Tx:     req.GetTxs()[i],
			Result: *res.GetTxResults()[i],
			Time:   &req.Time,
		}
		txResults = append(txResults, &txResult)
	}

	// index block
	eventNewBlockEvents := cmttypes.EventDataNewBlockEvents{
		Height: height,
		NumTxs: numTxs,
		Events: res.GetEvents(),
	}
	err := cs.es.IndexBlockEvents(eventNewBlockEvents)
	if err != nil {
		return fmt.Errorf("failed to index block, height: %d, err: %v", height, err)
	}

	// index txs
	err = cs.es.IndexTxEvents(txResults)
	if err != nil {
		return fmt.Errorf("failed to index block txs, height: %d, err: %v", height, err)
	}

	return nil
}

func (cs *TxEventSink) Tx(_ *rpctypes.Context, txHash string) (*ctypes.ResultTx, error) {
	if txHash == "" {
		return nil, fmt.Errorf("tx hash must not be empty")
	}

	txResponses, err := cs.GetTxByHash(txHash)
	if err != nil {
		return nil, err
	}

	return txResponses[0], nil
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

func (cs *TxEventSink) SearchTxs(q *cmtquery.Query, limit uint32) ([]*ctypes.ResultTx, uint64, error) {
	count := uint64(0)
	txResponses := []*ctypes.ResultTx{}

	conditions := q.Syntax()
	// conditions to skip because they're handled before "everything else"
	// If we are not matching events and tx.height = 3 occurs more than once, the later value will
	// overwrite the first one.
	conditions, heightInfo := DedupHeight(conditions)

	// extract ranges
	// if both upper and lower bounds exist, it's better to get them in order not
	// no iterate over kvs that are not within range.
	_, _, heightRange := cometbftindexer.LookForRangesWithHeight(conditions)
	heightInfo.HeightRange = heightRange
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
	-- filter txs based on input composite key conditions
	%s
	-- join everything to get the final table with sufficient data
	select
		tr.height,
		tr.created_at,
		tr.tx_hash,
		tr.tx_result
	from
		filtered_tx_event_attributes ftx
		join tx_results tr on tr.rowid = ftx.tx_id
	ORDER BY ftx.tx_id DESC;
	`, whereConditions, min(TxSearchLimit, limit), filterTableClause)

	// if there's no non-height condition -> we can simplify and optimize our query clause
	if len(filterArgs) == 0 {
		queryClause = fmt.Sprintf(`
		select distinct
			tr.height,
			tr.created_at,
			tr.tx_hash,
			tr.tx_result
		from
			tx_results tr 
		%s
		ORDER BY tr.height DESC
		LIMIT %d;
		`, whereConditions, min(TxSearchLimit, limit))
	}

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
	if cs.ri == nil {
		hclog.Default().Warn("Redpanda info is empty. Won't emit any events...")
		return nil
	}
	admin := cs.ri.GetAdmin()
	producer := cs.ri.GetProducer()

	for i, tx := range req.Txs {
		cosmosTx, err := indexerUtil.UnmarshalTxBz(cs, tx)
		if err != nil {
			return err
		}

		// get topic for tx
		var topicAndKeys []redpanda.TopicAndKey
		for _, message := range cosmosTx.Body.Messages {
			typeUrl := strings.Split(message.TypeUrl, "/")[1]
			typeUrlSplits := strings.Split(typeUrl, ".")

			var module, key string
			typeUrlLen := len(typeUrlSplits)
			// if typeUrl length is larger than 4 then is ibc message
			// else is cosmos message
			if typeUrlLen > 4 {
				module = typeUrlSplits[0]
			} else {
				module = typeUrlSplits[1]
			}

			key = typeUrlSplits[typeUrlLen-1]
			topic := "REDPANDA_TOPIC_" + strings.ToUpper(module)

			if !admin.IsTopicExist(topic) {
				err := admin.CreateTopic(topic)
				if err != nil {
					return err
				}

				cs.ri.SetTopics(module)
			}

			topicAndKeys = append(topicAndKeys, redpanda.TopicAndKey{Topic: topic, Key: key})
		}

		txHashBz := cmttypes.Tx(tx).Hash()
		topicMsg := ctypes.ResultTx{Height: req.Height, Hash: txHashBz, TxResult: *res.TxResults[i], Index: uint32(i), Tx: tx, Timestamp: req.Time.Format(time.RFC3339)}

		err = producer.SendToRedpanda(topicAndKeys, topicMsg)
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
SELECT height FROM ` + indexerCfg.TableBlocks + ` order by height desc limit 1;
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
	WHERE tx_results.tx_hash = '%s'`, indexerCfg.TableTxResults, txHash)).Scan(&height, &createdAt, &txResultBz); err != nil {
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
