package helpers

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"

	paramsutils "github.com/cosmos/cosmos-sdk/x/params/client/utils"
)

// ParamChangeProposal submits a param change proposal to the chain, signed by keyName.
func ParamChangeProposal(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	admin ibc.Wallet,
	prop *paramsutils.ParamChangeProposalJSON,
) (tx TxProposal, _ error) {
	tn := chain.GetNode()

	content, err := json.Marshal(prop)
	if err != nil {
		return tx, err
	}

	hash := sha256.Sum256(content)
	proposalFilename := fmt.Sprintf("%x.json", hash)
	err = tn.WriteFile(ctx, content, proposalFilename)
	if err != nil {
		return tx, fmt.Errorf("writing param change proposal: %w", err)
	}

	proposalPath := filepath.Join(tn.HomeDir(), proposalFilename)

	command := []string{
		"tokenfactory", "param-change", proposalPath,
	}

	txHash, err := tn.ExecTx(ctx, admin.KeyName(), command...)
	if err != nil {
		return tx, fmt.Errorf("failed to submit param change proposal: %w", err)
	}

	return txProposal(chain, txHash)
}

// QueryParam returns the state and details of a subspace param.
func QueryTokenFactoryParam(t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
) (TokenFactoryParams, error) {
	tn := chain.GetNode()
	stdout, _, err := tn.ExecQuery(ctx, "tokenfactory", "params")
	if err != nil {
		return TokenFactoryParams{}, err
	}
	var param QueryTokenFactoryParamsResponse
	err = json.Unmarshal(stdout, &param)
	if err != nil {
		return TokenFactoryParams{}, err
	}
	return param.Params, nil
}
