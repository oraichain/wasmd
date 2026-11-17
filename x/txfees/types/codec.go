package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgUpdateParams{}, "txfees/UpdateParams", nil)
	cdc.RegisterConcrete(&MsgAddFeeToken{}, "txfees/AddFeeToken", nil)
	cdc.RegisterConcrete(&MsgRemoveFeeToken{}, "txfees/RemoveFeeToken", nil)
	// NOTE: these names must stay byte-identical to the `option (amino.name)`
	// declared on the same message in proto/cosmwasm/txfees/v1/tx.proto. The
	// SIGN_MODE_LEGACY_AMINO_JSON handler in x/tx reads the name from the proto
	// option, so a divergence here produces sign bytes that do not match what a
	// Ledger / amino-json signer produced and signature verification fails.
	cdc.RegisterConcrete(&MsgAddBlacklist{}, "cosmwasm/x/txfees/MsgAddBlacklist", nil)
	cdc.RegisterConcrete(&MsgRemoveBlacklist{}, "cosmwasm/x/txfees/MsgRemoveBlacklist", nil)
	cdc.RegisterConcrete(&MsgRemoveBlacklistAndRecover{}, "cosmwasm/x/txfees/MsgRemoveBlacklistAndRecover", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
		&MsgAddFeeToken{},
		&MsgRemoveFeeToken{},
		&MsgAddBlacklist{},
		&MsgRemoveBlacklist{},
		&MsgRemoveBlacklistAndRecover{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
