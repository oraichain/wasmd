package helpers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
)

// Query helpers
// QueryParam returns the state and details of a subspace param.
func QueryAuthModuleAccounts(t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
) (AuthModuleAccounts, error) {
	tn := chain.GetNode()
	stdout, _, err := tn.ExecQuery(ctx, "auth", "module-accounts", "--output", "json")
	if err != nil {
		return AuthModuleAccounts{}, err
	}
	var accs AuthModuleAccounts
	err = json.Unmarshal(stdout, &accs)
	if err != nil {
		return AuthModuleAccounts{}, err
	}
	return accs, nil
}
