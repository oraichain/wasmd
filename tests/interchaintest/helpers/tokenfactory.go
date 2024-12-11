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
) (string, error) {
	tn := chain.GetNode()

	content, err := json.Marshal(prop)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(content)
	proposalFilename := fmt.Sprintf("%x.json", hash)
	err = tn.WriteFile(ctx, content, proposalFilename)
	if err != nil {
		return "", fmt.Errorf("writing param change proposal: %w", err)
	}

	proposalPath := filepath.Join(tn.HomeDir(), proposalFilename)

	command := []string{
		"tokenfactory", "param-change", proposalPath,
	}

	return tn.ExecTx(ctx, admin.KeyName(), command...)
}
