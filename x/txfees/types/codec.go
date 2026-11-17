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
	cdc.RegisterConcrete(&MsgAddBlacklist{}, "txfees/AddBlacklist", nil)
	cdc.RegisterConcrete(&MsgRemoveBlacklist{}, "txfees/RemoveBlacklist", nil)
	cdc.RegisterConcrete(&MsgRemoveBlacklistAndRecover{}, "txfees/RemoveBlacklistAndRecover", nil)
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
