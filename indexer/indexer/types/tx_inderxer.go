package types

import (
	"context"
	"errors"

	"github.com/cometbft/cometbft/libs/log"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/pubsub/query"
)

// Batch groups together multiple Index operations to be performed at the same time.
// NOTE: Batch is NOT thread-safe and must not be modified after starting its execution.
type Batch struct {
	Ops []*abci.TxResult
}

// NewBatch creates a new Batch.
func NewBatch(n int64) *Batch {
	return &Batch{
		Ops: make([]*abci.TxResult, n),
	}
}

// Add or update an entry for the given result.Index.
func (b *Batch) Add(result *abci.TxResult) error {
	b.Ops[result.Index] = result
	return nil
}

// Size returns the total number of operations inside the batch.
func (b *Batch) Size() int {
	return len(b.Ops)
}

// ErrorEmptyHash indicates empty hash
var ErrorEmptyHash = errors.New("transaction hash cannot be empty")

// TxIndexer interface defines methods to index and search transactions.
type TxIndexer interface {
	// AddBatch analyzes, indexes and stores a batch of transactions.
	AddBatch(b *Batch) error

	// Index analyzes, indexes and stores a single transaction.
	Index(result *abci.TxResult) error

	// Get returns the transaction specified by hash or nil if the transaction is not indexed
	// or stored.
	Get(hash []byte) (*abci.TxResult, error)

	// Search allows you to query for transactions.
	Search(ctx context.Context, q *query.Query) ([]*abci.TxResult, error)

	//Set Logger
	SetLogger(l log.Logger)
}

// PsqlTxIndexer implements the TxIndexer interface by delegating
// indexing operations to an underlying PostgreSQL event sink.
type PsqlTxIndexer struct{}

// AddBatch indexes a batch of transactions in Postgres, as part of TxIndexer.
func (p PsqlTxIndexer) AddBatch(batch Batch) error {
	// return p.psql.IndexTxEvents(batch.Ops)
	return nil
}

// Index indexes a single transaction result in Postgres, as part of TxIndexer.
func (p PsqlTxIndexer) Index(txr *abci.TxResult) error {
	// return p.psql.IndexTxEvents([]*abci.TxResult{txr})
	return nil
}

// Get is implemented to satisfy the TxIndexer interface, but is not supported
// by the psql event sink and reports an error for all inputs.
func (PsqlTxIndexer) Get([]byte) (*abci.TxResult, error) {
	return nil, errors.New("the TxIndexer.Get method is not supported")
}

// Search is implemented to satisfy the TxIndexer interface, but it is not
// supported by the psql event sink and reports an error for all inputs.
func (PsqlTxIndexer) Search(context.Context, *query.Query) ([]*abci.TxResult, error) {
	return nil, errors.New("the TxIndexer.Search method is not supported")
}

func (PsqlTxIndexer) SetLogger(log.Logger) {}
