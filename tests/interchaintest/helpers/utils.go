package helpers

import (
	"encoding/base64"
	"fmt"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
)

// TxProposal contains chain proposal transaction details.
type TxProposal struct {
	// The block height.
	Height int64
	// The transaction hash.
	TxHash string
	// Amount of gas charged to the account.
	GasSpent int64

	// Amount deposited for proposal.
	DepositAmount string
	// ID of proposal.
	ProposalID string
	// Type of proposal.
	ProposalType string
}

func txProposal(c *cosmos.CosmosChain, txHash string) (tx TxProposal, _ error) {
	txResp, err := c.GetTransaction(txHash)
	if err != nil {
		return tx, fmt.Errorf("failed to get transaction %s: %w", txHash, err)
	}
	tx.Height = txResp.Height
	tx.TxHash = txHash
	// In cosmos, user is charged for entire gas requested, not the actual gas used.
	tx.GasSpent = txResp.GasWanted
	events := txResp.Events

	tx.DepositAmount, _ = AttributeValue(events, "proposal_deposit", "amount")

	evtSubmitProp := "submit_proposal"
	tx.ProposalID, _ = AttributeValue(events, evtSubmitProp, "proposal_id")
	tx.ProposalType, _ = AttributeValue(events, evtSubmitProp, "proposal_type")

	return tx, nil
}

// AttributeValue returns an event attribute value given the eventType and attribute key tuple.
// In the event of duplicate types and keys, returns the first attribute value found.
// If not found, returns empty string and false.
func AttributeValue(events []abcitypes.Event, eventType, attrKey string) (string, bool) {
	for _, event := range events {
		if event.Type != eventType {
			continue
		}
		for _, attr := range event.Attributes {
			if attr.Key == attrKey {
				return attr.Value, true
			}

			// tendermint < v0.37-alpha returns base64 encoded strings in events.
			key, err := base64.StdEncoding.DecodeString(attr.Key)
			if err != nil {
				continue
			}
			if string(key) == attrKey {
				value, err := base64.StdEncoding.DecodeString(attr.Value)
				if err != nil {
					continue
				}
				return string(value), true
			}
		}
	}
	return "", false
}
