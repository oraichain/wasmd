package interchaintest

import (
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
)

func TestAddFeeToken(t *testing.T) {
	// set up testing env
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()
	chains := CreateChain(t, 1, 1)
	orai := chains[0].(*cosmos.CosmosChain)
	// add genesis account
	tn := orai.GetNode()
	ic, ctx := BuildInitialChainNoIbc(t, orai)
	t.Cleanup(func() {
		_ = ic.Close()
	})
	users := CreateTestingUser(t, ctx, t.Name(), genesisWalletAmount, chains...)
	oraiUser := users[0]
}
