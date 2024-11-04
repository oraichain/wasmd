package indexer

import (
	abci "github.com/cometbft/cometbft/abci/types"
)

type ModuleEventSinkIndexer interface {
	InsertModuleEvents(req *abci.RequestFinalizeBlock, res *abci.ResponseFinalizeBlock) error
	EmitModuleEvents(req *abci.RequestFinalizeBlock, res *abci.ResponseFinalizeBlock) error
	ModuleName() string
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
