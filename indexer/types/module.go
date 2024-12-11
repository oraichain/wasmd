package types

import (
	"github.com/CosmWasm/wasmd/app/params"
	"github.com/CosmWasm/wasmd/indexer/sink/psql"
	abci "github.com/cometbft/cometbft/abci/types"
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
	modules := make(map[string]ModuleEventSinkIndexer)
	manager := IndexerManager{Modules: modules}
	for _, indexer := range indexers {
		manager.Modules[indexer.ModuleName()] = indexer
	}
	return &manager
}
