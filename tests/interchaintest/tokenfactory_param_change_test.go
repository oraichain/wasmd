package interchaintest

import (
	"context"
	"encoding/json"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/oraichain/wasmd/tests/interchaintest/helpers"
	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testreporter"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	paramsutils "github.com/cosmos/cosmos-sdk/x/params/client/utils"
)

// TestStartOrai is a basic test to assert that spinning up a Orai network with 1 validator works properly.
func TestTokenfactoryParamChange(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	ctx := context.Background()

	numVals := 1
	numFullNodes := 1

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			Name:          "orai",
			ChainConfig:   oraiConfig,
			NumValidators: &numVals,
			NumFullNodes:  &numFullNodes,
		},
	})

	// Get chains from the chain factory
	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	orai := chains[0].(*cosmos.CosmosChain)

	client, network := interchaintest.DockerSetup(t)
	ic := interchaintest.NewInterchain().AddChain(orai)

	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)

	err = ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: false,

		// This can be used to write to the block database which will index all block data e.g. txs, msgs, events, etc.
		// BlockDatabaseFile: interchaintest.DefaultBlockDatabaseFilepath(),
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = ic.Close()
	})

	// Create some user accounts on both chains
	users := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), genesisWalletAmount, orai)

	// Wait a few blocks for relayer to start and for user accounts to be created
	err = testutil.WaitForBlocks(ctx, 5, orai)
	require.NoError(t, err)

	// Get our Bech32 encoded user addresses
	oraiUser := users[0]

	oldFeeParam, err := orai.QueryParam(ctx, "tokenfactory", "DenomCreationFee")
	require.NoError(t, err)
	_ = oldFeeParam

	paramChangeValue := sdk.NewCoins(sdk.NewInt64Coin("orai", 10_000_000))
	rawValue, err := json.Marshal(paramChangeValue)
	require.NoError(t, err)

	param_change := paramsutils.ParamChangeProposalJSON{
		Title:       ".",
		Description: ".",
		Changes: paramsutils.ParamChangesJSON{
			paramsutils.ParamChangeJSON{
				Subspace: "tokenfactory",
				Key:      "DenomCreationFee",
				Value:    rawValue,
			},
		},
		Deposit: "1000000000orai",
	}

	paramTx, err := helpers.ParamChangeProposal(t, ctx, orai, oraiUser, &param_change)
	require.NoError(t, err, "error submitting param change proposal tx")
	_ = paramTx
}
