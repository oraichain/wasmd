package interchaintest

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
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
func TestIbcHooks(t *testing.T) {
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
			Name:          "gaia",
			Version:       GaiaImageVersion,
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

	channel, err := ibc.GetTransferChannel(ctx, r, eRep, orai.Config().ChainID, gaia.Config().ChainID)
	require.NoError(t, err)

	// Create some user accounts on both chains
	users := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), genesisWalletAmount, orai, gaia)

	// Wait a few blocks for relayer to start and for user accounts to be created
	err = testutil.WaitForBlocks(ctx, 5, orai, gaia)
	require.NoError(t, err)

	// Get our Bech32 encoded user addresses
	oraiUser, gaiaUser := users[0], users[1]

	oraiUserAddress := sdk.MustBech32ifyAddressBytes(orai.Config().Bech32Prefix, oraiUser.Address())
	gaiaUserAddr := sdk.MustBech32ifyAddressBytes(gaia.Config().Bech32Prefix, gaiaUser.Address())

	_ = oraiUserAddress
	_ = gaiaUserAddr

	// Store and instantiate contract on Orai chain
	counterContractID, err := orai.StoreContract(ctx, oraiUser.KeyName(), "./bytecode/counter.wasm")
	require.NoError(t, err)

	initMsg := "{\"count\": 0}"
	contractAddress, err := orai.InstantiateContract(ctx, oraiUser.KeyName(), counterContractID, initMsg, true)
	require.NoError(t, err)

	// Get stake denom on orai
	gaiaDenom := transfertypes.GetPrefixedDenom(channel.Counterparty.PortID, channel.Counterparty.ChannelID, gaia.Config().Denom) //transfer/channel-0/uatom
	gaiaIBCDenom := transfertypes.ParseDenomTrace(gaiaDenom).IBCDenom()                                                           // ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2

	// check contract address balance
	balances, err := orai.BankQueryBalance(ctx, contractAddress, gaiaIBCDenom)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(0), balances)

	// send ibc transaction to execite the contract
	transfer := ibc.WalletAmount{
		Address: contractAddress,
		Denom:   gaia.Config().Denom,
		Amount:  amountToSend,
	}
	memo := fmt.Sprintf("{\"wasm\":{\"contract\":\"%s\",\"msg\": {\"increment\": {}} }}", contractAddress)
	transferTx, err := gaia.SendIBCTransfer(ctx, channel.Counterparty.ChannelID, gaiaUserAddr, transfer, ibc.TransferOptions{Memo: memo})
	require.NoError(t, err)

	// waiting for ACK -> transfer successfully
	gaiaHeight, err := gaia.Height(ctx)
	require.NoError(t, err)
	_, err = testutil.PollForAck(ctx, gaia, gaiaHeight-5, gaiaHeight+25, transferTx.Packet)
	require.NoError(t, err)

	// check new balances
	balances, err = orai.BankQueryBalance(ctx, contractAddress, gaiaIBCDenom)
	require.NoError(t, err)
	require.Equal(t, amountToSend, balances)

	// check contract
	var res GetCountResponse
	err = orai.QueryContract(ctx, contractAddress, QueryMsg{GetCount: &GetCountQuery{}}, &res)
	require.NoError(t, err)
	require.Equal(t, uint64(1), res.Data.Count)
}
