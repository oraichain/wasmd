package helpers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
)

// Query helpers
// QueryParam returns the state and details of a subspace param.
func QueryBankDenomMetadata(t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	denom string,
) (QueryDenomMetadataResponse, error) {
	tn := chain.GetNode()
	stdout, _, err := tn.ExecQuery(ctx, "bank", "denom-metadata", denom, "--output", "json")
	if err != nil {
		return QueryDenomMetadataResponse{}, err
	}
	var metadata QueryDenomMetadataResponse
	err = json.Unmarshal(stdout, &metadata)
	if err != nil {
		return QueryDenomMetadataResponse{}, err
	}
	return metadata, nil
}
