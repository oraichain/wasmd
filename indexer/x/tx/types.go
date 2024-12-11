package tx

import txtypes "github.com/cosmos/cosmos-sdk/types/tx"

// Result of searching for txs
type ResultTxSearch struct {
	Txs        []*txtypes.GetTxResponse `json:"txs"`
	TotalCount uint64                   `json:"total_count"`
}
