package indexer

import (
	"github.com/CosmWasm/wasmd/app/params"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/state/indexer/sink/psql"
)

type ModuleEventSinkIndexer interface {
	InsertModuleEvents(req *abci.RequestFinalizeBlock, res *abci.ResponseFinalizeBlock) error
	EmitModuleEvents(req *abci.RequestFinalizeBlock, res *abci.ResponseFinalizeBlock) error
	ModuleName() string
	EventSink() *psql.EventSink
	EncodingConfig() params.EncodingConfig
}

type IndexerManager struct {
	Modules map[string]ModuleEventSinkIndexer
}

func NewIndexerManager(indexers ...ModuleEventSinkIndexer) *IndexerManager {
	manager := IndexerManager{}
	for _, indexer := range indexers {
		manager.Modules[indexer.ModuleName()] = indexer
	}
	return &manager
}
