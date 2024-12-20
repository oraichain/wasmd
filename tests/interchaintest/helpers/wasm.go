package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
)

// Proposal helpers
func ProposalSetGasLessContracts(
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
		Title:    "Set gas less contracts",
		Summary:  "Set gas less contracts",
	}

	message := wasmtypes.MsgSetGaslessContracts{
		Authority: sdk.MustBech32ifyAddressBytes(chain.Config().Bech32Prefix, authtypes.NewModuleAddress(govtypes.ModuleName)),
		Contracts: contractAddress,
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

func ProposalUnsetGasLessContracts(
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
		Title:    "Unset gas less contracts",
		Summary:  "Unset gas less contracts",
	}

	message := wasmtypes.MsgUnsetGaslessContracts{
		Authority: sdk.MustBech32ifyAddressBytes(chain.Config().Bech32Prefix, authtypes.NewModuleAddress(govtypes.ModuleName)),
		Contracts: contractAddress,
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
func QueryGasLessContracts(
	ctx context.Context,
	chain *cosmos.CosmosChain,
) ([]string, error) {
	tn := chain.GetNode()
	stdout, _, err := tn.ExecQuery(ctx, "wasm", "gas-less-contracts")
	if err != nil {
		return []string{}, err
	}
	var res QueryWasmGasLessContracts
	err = json.Unmarshal(stdout, &res)
	if err != nil {
		return []string{}, err
	}
	return res.ContractAddresses, nil
}
