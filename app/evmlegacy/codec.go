package evmlegacy

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	evmtypes "github.com/cosmos/evm/x/vm/types"
)

// RegisterInterfaces registers decode-only stubs for EVM message types that remain
// in historical state (e.g. gov proposal #316 embeds MsgUpdateParams) after the EVM
// AppModule was soft-removed.
//
// Intentionally does NOT call msgservice.RegisterMsgServiceDesc — txs carrying these
// type URLs still have no msg service route and cannot execute.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&evmtypes.MsgUpdateParams{},
	)
}
