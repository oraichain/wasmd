package helpers

import (
	"context"
	"fmt"
	"strconv"

	txfeestypes "github.com/CosmWasm/wasmd/x/txfees/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
)

var (
	USDAI = "factory/orai1wuvhex9xqs3r539mvc6mtm7n20fcj3qr2m0y9khx6n5vtlngfzes3k0rq9/DYeTA4ZQhEwoJ5imjq1Q3zgwfTgkh4WmdfFHAq3jLrv3"
)

func ProposalAddFeeToken(
	ctx context.Context,
	chain *cosmos.CosmosChain,
	user ibc.Wallet,
	contractAddress []string,
	deposit sdk.Coin,
	gas uint64,
) (uint64, error) {
	tn := chain.GetNode()

	proposal := cosmos.TxProposalv1{
		Metadata: "none",
		Deposit:  deposit.String(),
		Title:    "add fee token",
		Summary:  "add fee token",
	}

	message := txfeestypes.MsgAddFeeToken{
		Authority: sdk.MustBech32ifyAddressBytes(chain.Config().Bech32Prefix, authtypes.NewModuleAddress(govtypes.ModuleName)),
		Config: txfeestypes.FeeTokenConfiguration{
			Denom:  USDAI,
			PoolId: "",
			Status: txfeestypes.FeeTokenStatus_FROZEN,
		},
	}

	msg, err := chain.Config().EncodingConfig.Codec.MarshalInterfaceJSON(&message)
	if err != nil {
		return 0, err
	}
	proposal.Messages = append(proposal.Messages, msg)

	txHash, err := tn.SubmitProposal(ctx, user.KeyName(), proposal)
	if err != nil {
		return 0, err
	}

	txProposal, err := txProposal(chain, txHash)
	if err != nil {
		return 0, fmt.Errorf("failed to parse tx proposal information: %w", err)
	}

	propId, err := strconv.ParseUint(txProposal.ProposalID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse tx proposal information proposal ID: %w", err)
	}

	return propId, nil
}
