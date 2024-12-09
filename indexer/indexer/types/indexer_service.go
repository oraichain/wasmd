package types

import (
	"github.com/CosmWasm/wasmd/indexer/indexer/sink/psql"
	abci "github.com/cometbft/cometbft/abci/types"
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
	return nil
}
