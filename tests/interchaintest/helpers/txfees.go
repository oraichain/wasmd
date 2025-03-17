package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	txfeestypes "github.com/CosmWasm/wasmd/x/txfees/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
)

func ProposalTxfeesUpdateParams(
	ctx context.Context,
	chain *cosmos.CosmosChain,
	user ibc.Wallet,
	params txfeestypes.Params,
	deposit sdk.Coin,
) (uint64, error) {
	tn := chain.GetNode()

	proposal := cosmos.TxProposalv1{
		Metadata: "none",
		Deposit:  deposit.String(),
		Title:    "update params",
		Summary:  "update params",
	}

	message := txfeestypes.MsgUpdateParams{
		Authority: sdk.MustBech32ifyAddressBytes(chain.Config().Bech32Prefix, authtypes.NewModuleAddress(govtypes.ModuleName)),
		Params:    params,
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

func ProposalTxfeesAddFeeToken(
	ctx context.Context,
	chain *cosmos.CosmosChain,
	user ibc.Wallet,
	denom string,
	deposit sdk.Coin,
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
		Denom:     denom,
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

// Query helpers
func QueryTokenExchangeRate(
	ctx context.Context,
	chain *cosmos.CosmosChain,
	denom string,
) (string, error) {
	tn := chain.GetNode()
	stdout, _, err := tn.ExecQuery(ctx, "txfees", "token-exchange-rate", denom)
	if err != nil {
		return "", err
	}
	var res QueryTxfeesTokenExchangeRate
	err = json.Unmarshal(stdout, &res)
	if err != nil {
		return "", err
	}
	return res.Rate, nil
}

// BankSend sends tokens from one account to another.
func BankSend(ctx context.Context, chain *cosmos.CosmosChain, keyName string, amount ibc.WalletAmount, fee sdk.Coin) error {
	tn := chain.GetNode()
	_, err := tn.ExecTx(ctx,
		keyName, "bank", "send", keyName,
		amount.Address, fmt.Sprintf("%s%s", amount.Amount.String(), amount.Denom),
		"--fees", fee.String(),
	)
	return err
}
