package types

import (
	"context"
	"errors"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/libs/pubsub/query"
	"github.com/cometbft/cometbft/types"
)

// BlockIndexer defines an interface contract for indexing block events.
type BlockIndexer interface {
	// Has returns true if the given height has been indexed. An error is returned
	// upon database query failure.
	Has(height int64) (bool, error)

	// Index indexes FinalizeBlock events for a given block by its height.
	Index(types.EventDataNewBlockEvents) error

	// Search performs a query for block heights that match a given FinalizeBlock
	// event search criteria.
	Search(ctx context.Context, q *query.Query) ([]int64, error)

	SetLogger(l log.Logger)
}

// PsqlBlockIndexer implements the BlockIndexer interface by
// delegating indexing operations to an underlying PostgreSQL event sink.
type PsqlBlockIndexer struct {
}

// Has is implemented to satisfy the BlockIndexer interface, but it is not
// supported by the psql event sink and reports an error for all inputs.
func (PsqlBlockIndexer) Has(_ int64) (bool, error) {
	return false, errors.New("the BlockIndexer.Has method is not supported")
}

// Index indexes block begin and end events for the specified block.  It is
// part of the BlockIndexer interface.
func (p PsqlBlockIndexer) Index(block types.EventDataNewBlockEvents) error {
	return nil
	// return p.psql.IndexBlockEvents(block)
}

// Search is implemented to satisfy the BlockIndexer interface, but it is not
// supported by the psql event sink and reports an error for all inputs.
func (PsqlBlockIndexer) Search(context.Context, *query.Query) ([]int64, error) {
	return nil, errors.New("the BlockIndexer.Search method is not supported")
}

func (PsqlBlockIndexer) SetLogger(log.Logger) {}
