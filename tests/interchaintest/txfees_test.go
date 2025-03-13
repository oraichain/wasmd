package interchaintest

import (
	"fmt"
	"testing"

	"github.com/oraichain/wasmd/tests/interchaintest/helpers"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/stretchr/testify/require"
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
	ic, ctx := BuildInitialChainNoIbc(t, orai)
	t.Cleanup(func() {
		_ = ic.Close()
	})
	users := CreateTestingUser(t, ctx, t.Name(), genesisWalletAmount, chains...)
	oraiUser := users[0]

	// create new token
	expectedDenom, _ := helpers.TxTokenFactoryCreateDenom(t, ctx, orai, oraiUser, "usdai", 100_000_000)
	denomCreated, err := helpers.QueryDenomsFromCreator(t, ctx, orai, oraiUser.FormattedAddress())
	require.NoError(t, err)
	require.Contains(t, denomCreated, expectedDenom)

	authorityAdmin, err := helpers.QueryDenomAuthorityMetadata(t, ctx, orai, expectedDenom)
	require.NoError(t, err)
	require.Equal(t, oraiUser.FormattedAddress(), authorityAdmin)

	// mint token
	tokenToMint := uint64(100_000_000_000)
	_ = helpers.TxTokenFactoryMintToken(t, ctx, orai, oraiUser, expectedDenom, tokenToMint)
	oraiUserBalance, err := helpers.QueryBalance(t, ctx, orai, expectedDenom, oraiUser.FormattedAddress())
	require.NoError(t, err)
	require.Equal(t, tokenToMint, oraiUserBalance)

	// setup price contract with new token
	// Store and instantiate contract on Orai chain
	// mock oraidex v3
	mockDexV3ContractId, err := orai.StoreContract(ctx, oraiUser.KeyName(), "./bytecode/mock-oraidex-v3.wasm")
	require.NoError(t, err)
	initMsg := "{}"
	mockDexV3Address, err := orai.InstantiateContract(ctx, oraiUser.KeyName(), mockDexV3ContractId, initMsg, true)
	require.NoError(t, err)

	// price contract
	priceContractId, err := orai.StoreContract(ctx, oraiUser.KeyName(), "./bytecode/price-query-local.wasm")
	require.NoError(t, err)
	initMsg = fmt.Sprintf(`{"base_token": "orai", "oraidex_v3_addr": "%s"}`, mockDexV3Address)
	priceContractAddress, err := orai.InstantiateContract(ctx, oraiUser.KeyName(), priceContractId, initMsg, true)
	require.NoError(t, err)

	exeMsg := fmt.Sprintf(`{"add_pool": {"quote_token": "%s","fee_tier": {"fee": 3000000000, "tick_spacing": 100}}}`, expectedDenom)
	_, err = orai.ExecuteContract(ctx, oraiUser.KeyName(), priceContractAddress, exeMsg, "--gas", "auto")
	require.NoError(t, err)

	var res helpers.QuerySqrtPriceResponse
	queryMsg := fmt.Sprintf(`{"get_sqrt_price": {"quote_token": "%s"}}`, expectedDenom)
	err = orai.QueryContract(ctx, priceContractAddress, queryMsg, &res)
	require.NoError(t, err)

	fmt.Println(res.Data)
}
