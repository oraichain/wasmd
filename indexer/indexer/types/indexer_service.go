package types

import (
	"fmt"

	"github.com/CosmWasm/wasmd/indexer/indexer/sink/psql"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtTypes "github.com/cometbft/cometbft/types"
)

type IndexerService struct {
	txIndexer    TxIndexer
	blockIndexer BlockIndexer
}

func NewIndexerService(es *psql.EventSink) *IndexerService {
	txIndexer := NewTxIndexer(es)
	blockIndexer := NewBlockIndexer(es)

	return &IndexerService{txIndexer: txIndexer, blockIndexer: blockIndexer}
}

func (is *IndexerService) IndexBlockAndTxs(req *abci.RequestFinalizeBlock, res *abci.ResponseFinalizeBlock) error {
	height := req.GetHeight()
	numTxs := int64(len(req.GetTxs()))

	batch := NewBatch(numTxs)
	for i := int64(0); i < numTxs; i++ {
		txResult := abci.TxResult{
			Height: height,
			Index:  uint32(i),
			Tx:     req.GetTxs()[i],
			Result: *res.GetTxResults()[i],
			Time:   &req.Time,
		}

		err := batch.Add(&txResult)
		if err != nil {
			return fmt.Errorf("failed to add tx to batch, height: %d, index: %d, err: %v", height, i, err)
		}
	}

	// index block
	eventNewBlockEvents := cmtTypes.EventDataNewBlockEvents{
		Height: height,
		NumTxs: numTxs,
		Events: res.GetEvents(),
	}
	err := is.blockIndexer.Index(eventNewBlockEvents)
	if err != nil {
		return fmt.Errorf("failed to index block, height: %d, err: %v", height, err)
	}

	// index txs
	err = is.txIndexer.AddBatch(batch)
	if err != nil {
		return fmt.Errorf("failed to index block txs, height: %d, err: %v", height, err)
	}

	return nil
}
