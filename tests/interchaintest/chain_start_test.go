package interchaintest

import (
	"context"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestStartOrai is a basic test to assert that spinning up a Orai network with 1 validator works properly.
func TestStartOrai(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

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

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	orai := chains[0].(*cosmos.CosmosChain)
	ic := interchaintest.NewInterchain().AddChain(orai)
	ctx := context.Background()
	client, network := interchaintest.DockerSetup(t)

	require.NoError(t, ic.Build(ctx, nil, interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: true,
	}))
	t.Cleanup(func() {
		_ = ic.Close()
	})

	a, err := orai.AuthQueryModuleAccounts(ctx)
	require.NoError(t, err)
	t.Log("module accounts", a)
}
