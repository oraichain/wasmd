package interchaintest

import (
	"encoding/json"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/oraichain/wasmd/tests/interchaintest/helpers"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/stretchr/testify/require"
)

func TestWasmGasLessContract(t *testing.T) {
	// set up testing env
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()
	chains := CreateChain(t, 1, 1)
	orai := chains[0].(*cosmos.CosmosChain)
	ic, ctx := BuildInitialChainNoIbc(t, orai)
	t.Cleanup(func() {
		_ = ic.Close()
	})
	users := CreateTestingUser(t, ctx, t.Name(), genesisWalletAmount, chains...)
	oraiUser := users[0]

	// Store and instantiate contract on Orai chain
	counterContractID, err := orai.StoreContract(ctx, oraiUser.KeyName(), "./bytecode/hackatom.wasm")
	require.NoError(t, err)

	initMsg := helpers.HackatomExampleInitMsg{
		Verifier:    oraiUser.FormattedAddress(),
		Beneficiary: oraiUser.FormattedAddress(),
	}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)
	contractAddress, err := orai.InstantiateContract(ctx, oraiUser.KeyName(), counterContractID, string(initMsgBz), true)
	require.NoError(t, err)

	// Execute contract
	executeMsg := `{"release":{}}`
	resBefore, err := orai.ExecuteContract(ctx, oraiUser.KeyName(), contractAddress, executeMsg, "--gas", "auto")
	require.NoError(t, err)

	// Test set gas less contract successfully
	// Submit set gas less contract proposal
	proposalSetGasLessID, err := helpers.ProposalSetGasLessContracts(
		ctx,
		orai,
		oraiUser,
		[]string{contractAddress},
		sdk.NewCoin(orai.Config().Denom, math.NewIntFromUint64(100_000_000)),
		1_000_000,
	)
	require.NoError(t, err)

	// vote on created proposal and waiting for passed proposal
	err = orai.VoteOnProposalAllValidators(ctx, proposalSetGasLessID, cosmos.ProposalVoteYes)
	require.NoError(t, err, "failed to submit votes")
	height, _ := orai.Height(ctx)
	_, err = cosmos.PollForProposalStatus(ctx, orai, height, height+10, proposalSetGasLessID, govv1beta1.StatusPassed)
	require.NoError(t, err, "proposal status did not change to passed in expected number of blocks")

	// Check gas less contract
	gasLessContractsBefore, err := helpers.QueryGasLessContracts(ctx, orai)
	require.NoError(t, err)
	require.Contains(t, gasLessContractsBefore, contractAddress)

	// re execute
	resAfter, err := orai.ExecuteContract(ctx, oraiUser.KeyName(), contractAddress, executeMsg, "--gas", "auto")
	require.NoError(t, err)
	require.Less(t, resAfter.GasUsed, resBefore.GasUsed) // after set gas less gas used should be less than before

	// Test unset gas less contract successfully
	proposalUnsetGasLessID, err := helpers.ProposalUnsetGasLessContracts(
		ctx,
		orai,
		oraiUser,
		[]string{contractAddress},
		sdk.NewCoin(orai.Config().Denom, math.NewIntFromUint64(100_000_000)),
		1_000_000,
	)
	require.NoError(t, err)

	// vote on created proposal and waiting for passed proposal
	err = orai.VoteOnProposalAllValidators(ctx, proposalUnsetGasLessID, cosmos.ProposalVoteYes)
	require.NoError(t, err, "failed to submit votes")
	height, _ = orai.Height(ctx)
	_, err = cosmos.PollForProposalStatus(ctx, orai, height, height+10, proposalUnsetGasLessID, govv1beta1.StatusPassed)
	require.NoError(t, err, "proposal status did not change to passed in expected number of blocks")

	// Check gas less contract
	gasLessContractsAfter, err := helpers.QueryGasLessContracts(ctx, orai)
	require.NoError(t, err)
	require.Empty(t, gasLessContractsAfter)
}
