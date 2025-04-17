package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
)

func QueryBankBalance(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	denom string,
	userAddress string,
) (uint64, error) {
	tn := chain.GetNode()
	stdout, _, err := tn.ExecQuery(ctx, "bank", "balance", userAddress, denom)
	if err != nil {
		return 0, err
	}
	fmt.Println("Bank balance: ", string(stdout))
	if stdout == nil {
		return 0, err
	}

	var balance QueryBalanceResponse
	err = json.Unmarshal(stdout, &balance)
	if err != nil {
		return 0, err
	}

	return balance.Balance.Amount.Uint64(), nil
}

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
