package helpers

import (
	"context"
	"encoding/json"
	"testing"

	// govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	// "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
)

// Query helpers
// QueryParam returns the state and details of a subspace param.
func QueryGovProposalStatus(t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	propId string,
) (QueryProposalResponse, error) {
	tn := chain.GetNode()
	stdout, _, err := tn.ExecQuery(ctx, "gov", "proposal", propId, "--output", "json")
	if err != nil {
		return QueryProposalResponse{}, err
	}
	var proposal QueryProposalResponse
	err = json.Unmarshal(stdout, &proposal)
	if err != nil {
		return QueryProposalResponse{}, err
	}
	return proposal, nil
}
