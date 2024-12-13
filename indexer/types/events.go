package types

import (
	abci "github.com/cometbft/cometbft/abci/types"
)

type EventDataNewBlockEvents struct {
	Height int64        `json:"height"`
	Events []abci.Event `json:"events"`
	NumTxs int64        `json:"num_txs,string"` // Number of txs in a block
}
