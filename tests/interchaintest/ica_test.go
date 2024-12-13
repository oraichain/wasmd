package interchaintest

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	"github.com/oraichain/wasmd/tests/interchaintest/helpers"
	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/relayer"
	"github.com/strangelove-ventures/interchaintest/v8/testreporter"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestStartOrai is a basic test to assert that spinning up a Orai network with 1 validator works properly.
func TestInterchainAccount(t *testing.T) {
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
		{
			Name:    "gaia",
			Version: GaiaImageVersion,
			ChainConfig: ibc.ChainConfig{
				GasPrices: "1uatom",
			},
			NumValidators: &numVals,
			NumFullNodes:  &numFullNodes,
		},
	})

	// Get chains from the chain factory
	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	orai, gaia := chains[0].(*cosmos.CosmosChain), chains[1].(*cosmos.CosmosChain)

	// Create relayer factory to utilize the go-relayer
	client, network := interchaintest.DockerSetup(t)
	r := interchaintest.NewBuiltinRelayerFactory(
		ibc.CosmosRly,
		zaptest.NewLogger(t),
		relayer.CustomDockerImage(IBCRelayerImage, IBCRelayerVersion, "100:1000"),
	).Build(t, client, network)

	ic := interchaintest.NewInterchain().
		AddChain(orai).
		AddChain(gaia).
		AddRelayer(r, "rly").
		AddLink(interchaintest.InterchainLink{
			Chain1:  orai,
			Chain2:  gaia,
			Relayer: r,
			Path:    pathOraiGaia,
		})

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

	// Start the relayer
	require.NoError(t, r.StartRelayer(ctx, eRep, pathOraiGaia))
	t.Cleanup(
		func() {
			err := r.StopRelayer(ctx, eRep)
			if err != nil {
				panic(fmt.Errorf("an error occurred while stopping the relayer: %s", err))
			}
		},
	)

	// Fund testing user
	users := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), genesisWalletAmount, orai, gaia)
	// Get our Bech32 encoded user addresses
	oraiUser, gaiaUser := users[0], users[1]

	relayerWalletOrai, found := r.GetWallet(orai.Config().ChainID)
	require.True(t, found)

	err = orai.SendFunds(ctx, oraiUser.KeyName(), ibc.WalletAmount{
		Address: relayerWalletOrai.FormattedAddress(),
		Amount:  math.NewInt(100_000_000),
		Denom:   orai.Config().Denom,
	})
	require.NoError(t, err)

	relayerWalletGaia, found := r.GetWallet(gaia.Config().ChainID)
	require.True(t, found)

	err = gaia.SendFunds(ctx, gaiaUser.KeyName(), ibc.WalletAmount{
		Address: relayerWalletGaia.FormattedAddress(),
		Amount:  math.NewInt(100_000_000),
		Denom:   gaia.Config().Denom,
	})
	require.NoError(t, err)

	// Get ibc connection
	ibcConnection, err := r.GetConnections(ctx, eRep, orai.Config().ChainID)
	require.NoError(t, err)

	// register ICA
	_, err = helpers.RegisterICA(t, ctx, orai, oraiUser.KeyName(), ibcConnection[0].ID)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 10, orai, gaia)

	require.NoError(t, err)
	icaAddress, err := helpers.QueryInterchainAccount(t, ctx, orai, oraiUser.FormattedAddress(), ibcConnection[0].ID)
	require.NoError(t, err)
	require.NotEmpty(t, icaAddress)
}
