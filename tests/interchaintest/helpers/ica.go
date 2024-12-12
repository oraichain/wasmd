package helpers

import (
	"context"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
)

// RegisterICA will attempt to register an interchain account on the counterparty chain.
func RegisterICA(t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	keyName, connectionID string,
) (string, error) {
	tn := chain.GetNode()
	return tn.ExecTx(ctx, keyName,
		"interchain-accounts",
		"controller",
		"register",
		connectionID,
		"--version", "",
		"--gas", "auto",
	)
}
