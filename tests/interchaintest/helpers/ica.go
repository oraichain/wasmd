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

// QueryParam returns the state and details of a subspace param.
func QueryInterchainAccount(t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	owner string,
	connectionID string,
) (string, error) {
	tn := chain.GetNode()
	stdout, _, err := tn.ExecQuery(
		ctx,
		"interchain-accounts",
		"controller",
		"interchain-account",
		owner,
		connectionID,
	)
	if err != nil {
		return "", err
	}
	return string(stdout), nil
}
