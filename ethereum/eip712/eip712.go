package eip712

import (
	"encoding/json"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	evmtypes "github.com/CosmWasm/wasmd/x/evm/types"
)

// ConstructUntypedEIP712Data returns the bytes to sign for a transaction.
func ConstructUntypedEIP712Data(chainID string, accnum, sequence, timeout uint64, fee legacytx.StdFee, msgs []sdk.Msg, memo string) []byte {
	signBytes := legacytx.StdSignBytes(chainID, accnum, sequence, timeout, fee, msgs, memo)
	var inInterface map[string]interface{}
	err := json.Unmarshal(signBytes, &inInterface)
	if err != nil {
		panic(err)
	}

	// remove msgs from the sign doc since we will be adding them as separate fields
	delete(inInterface, "msgs")

	// Add messages as separate fields
	for i := 0; i < len(msgs); i++ {
		msg := msgs[i]
		legacyMsg, ok := msg.(legacytx.LegacyMsg)
		if !ok {
			panic(fmt.Errorf("expected %T when using amino JSON", (*legacytx.LegacyMsg)(nil)))
		}
		msgsBytes := json.RawMessage(legacyMsg.GetSignBytes())
		inInterface[fmt.Sprintf("msg%d", i+1)] = msgsBytes
	}

	bz, err := json.Marshal(inInterface)
	if err != nil {
		panic(err)
	}
	return sdk.MustSortJSON(bz)
}

// ComputeTypedDataHash computes keccak hash of typed data for signing.
func ComputeTypedDataHash(typedData apitypes.TypedData) ([]byte, error) {
	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		err = errorsmod.Wrap(err, "failed to pack and hash typedData EIP712Domain")
		return nil, err
	}

	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		err = errorsmod.Wrap(err, "failed to pack and hash typedData primary type")
		return nil, err
	}

	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(typedDataHash)))
	return crypto.Keccak256(rawData), nil
}

// WrapTxToTypedData is an ultimate method that wraps Amino-encoded Cosmos Tx JSON data
// into an EIP712-compatible TypedData request.
func WrapTxToTypedData(
	chainID uint64,
	msgs []sdk.Msg,
	data []byte,
	feeDelegation *FeeDelegationOptions,
	params evmtypes.Params,
) (apitypes.TypedData, error) {
	txData := make(map[string]interface{})

	if err := json.Unmarshal(data, &txData); err != nil {
		return apitypes.TypedData{}, errorsmod.Wrap(sdkerrors.ErrJSONUnmarshal, "failed to JSON unmarshal data")
	}

	domain := getTypedDataDomain(chainID)

	msgTypes, err := extractMsgTypes(msgs, params)
	if err != nil {
		return apitypes.TypedData{}, err
	}

	if feeDelegation != nil {
		feeInfo, ok := txData["fee"].(map[string]interface{})
		if !ok {
			return apitypes.TypedData{}, errorsmod.Wrap(sdkerrors.ErrInvalidType, "cannot parse fee from tx data")
		}

		feeInfo["feePayer"] = feeDelegation.FeePayer.String()

		// also patching msgTypes to include feePayer
		msgTypes["Fee"] = []apitypes.Type{
			{Name: "feePayer", Type: "string"},
			{Name: "amount", Type: "Coin[]"},
			{Name: "gas", Type: "string"},
		}
	}

	typedData := apitypes.TypedData{
		Types:       msgTypes,
		PrimaryType: "Tx",
		Domain:      domain,
		Message:     txData,
	}

	return typedData, nil
}

type FeeDelegationOptions struct {
	FeePayer sdk.AccAddress
}

func extractMsgTypes(msgs []sdk.Msg, params evmtypes.Params) (apitypes.Types, error) {
	rootTypes := getRootTypes()

	// Add types each message
	for i := 0; i < len(msgs); i++ {
		msg := msgs[i]
		msgAttrName := fmt.Sprintf("msg%d", i+1)
		msgTypeName := fmt.Sprintf("Msg%d", i+1)

		// ensure eip712 messages implement legacytx.LegacyMsg
		_, ok := msg.(legacytx.LegacyMsg)
		if !ok {
			err := errorsmod.Wrapf(sdkerrors.ErrInvalidType, "msg %T must implement legacytx.LegacyMsg", (*legacytx.LegacyMsg)(nil))
			return apitypes.Types{}, err
		}

		// get corresponding allowed msg from params
		msgType := sdk.MsgTypeURL(msg)
		allowedMsg := params.EIP712AllowedMsgFromMsgType(msgType)
		if allowedMsg == nil {
			err := errorsmod.Wrapf(
				sdkerrors.ErrInvalidType,
				"eip712 message type \"%s\" is not permitted",
				msgType,
			)
			return apitypes.Types{}, err
		}

		// Add msg property to tx
		txMsgType := apitypes.Type{Name: msgAttrName, Type: msgTypeName}
		rootTypes["Tx"] = append(rootTypes["Tx"], txMsgType)

		// Add msg type to root types
		msgValueTypeName := allowedMsg.MsgValueTypeName
		rootTypes[msgTypeName] = []apitypes.Type{
			{Name: "type", Type: "string"},
			{Name: "value", Type: msgValueTypeName},
		}

		// Add msg value type and nested types
		if rootTypes[msgValueTypeName] == nil && allowedMsg != nil {
			// add msg value type
			rootTypes[msgValueTypeName] = msgAttrsToEIP712Types(allowedMsg.ValueTypes)

			// add nested types
			for _, nestedType := range allowedMsg.NestedTypes {
				nestedTypeName := nestedType.Name
				if rootTypes[nestedTypeName] == nil {
					rootTypes[nestedTypeName] = msgAttrsToEIP712Types(nestedType.Attrs)
				}
			}
		}
	}
	return rootTypes, nil
}

// msgAttrsToEIP712Types converts a slice of EIP712MsgAttrType to a slice of apitypes.Type.
func msgAttrsToEIP712Types(attrTypes []evmtypes.EIP712MsgAttrType) []apitypes.Type {
	msgTypes := make([]apitypes.Type, len(attrTypes))
	for i, attrType := range attrTypes {
		apitypes := apitypes.Type{
			Name: attrType.Name,
			Type: attrType.Type,
		}
		msgTypes[i] = apitypes
	}
	return msgTypes
}
