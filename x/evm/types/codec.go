package types

import (
	errorsmod "cosmossdk.io/errors"
	"github.com/CosmWasm/wasmd/crypto/ethsecp256k1"
	"github.com/CosmWasm/wasmd/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	proto "github.com/cosmos/gogoproto/proto"
)

var ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

type (
	ExtensionOptionsEthereumTxI interface{}
)

// RegisterInterfaces registers the client interfaces to protobuf Any.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgEthereumTx{},
		&MsgSetMappingEvmAddress{},
		// &MsgDeleteMappingEvmAddress{},
	)
	registry.RegisterInterface(
		"ethermint.evm.v1.ExtensionOptionsEthereumTx",
		(*ExtensionOptionsEthereumTxI)(nil),
		&ExtensionOptionsEthereumTx{},
	)
	registry.RegisterInterface(
		"ethermint.evm.v1.TxData",
		(*TxData)(nil),
		&DynamicFeeTx{},
		&AccessListTx{},
		&LegacyTx{},
	)

	registry.RegisterInterface("cosmwasm.types.v1.EthAccount", (*types.EthAccountI)(nil), &types.EthAccount{})
	registry.RegisterInterface("cosmwasm.crypto.v1.ethsecp256k1.PubKey", (*cryptotypes.PubKey)(nil), &ethsecp256k1.PubKey{})
	registry.RegisterInterface("cosmwasm.crypto.v1.ethsecp256k1.PrivKey", (*cryptotypes.PrivKey)(nil), &ethsecp256k1.PrivKey{})

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

// PackClientState constructs a new Any packed with the given tx data value. It returns
// an error if the client state can't be casted to a protobuf message or if the concrete
// implemention is not registered to the protobuf codec.
func PackTxData(txData TxData) (*codectypes.Any, error) {
	msg, ok := txData.(proto.Message)
	if !ok {
		return nil, errorsmod.Wrapf(sdkerrors.ErrPackAny, "cannot proto marshal %T", txData)
	}

	anyTxData, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrPackAny, err.Error())
	}

	return anyTxData, nil
}

// UnpackTxData unpacks an Any into a TxData. It returns an error if the
// client state can't be unpacked into a TxData.
func UnpackTxData(any *codectypes.Any) (TxData, error) {
	if any == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnpackAny, "protobuf Any message cannot be nil")
	}

	txData, ok := any.GetCachedValue().(TxData)
	if !ok {
		return nil, errorsmod.Wrapf(sdkerrors.ErrUnpackAny, "cannot unpack Any into TxData %T", any)
	}

	return txData, nil
}
