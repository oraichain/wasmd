package eip712

import (
	"encoding/json"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/CosmWasm/wasmd/tests"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingTypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/stretchr/testify/require"

	evmtypes "github.com/CosmWasm/wasmd/x/evm/types"
)

func TestExtractMsgTypes(t *testing.T) {
	params := evmtypes.DefaultParams()
	params.EIP712AllowedMsgs = []evmtypes.EIP712AllowedMsg{
		{
			MsgTypeUrl:       "/cosmos.bank.v1beta1.MsgSend",
			MsgValueTypeName: "MsgValueSend",
			ValueTypes: []evmtypes.EIP712MsgAttrType{
				{Name: "from_address", Type: "string"},
				{Name: "to_address", Type: "string"},
				{Name: "amount", Type: "Coin[]"},
			},
		},
		{
			MsgTypeUrl:       "/cosmos.staking.v1beta1.MsgDelegate",
			MsgValueTypeName: "MsgValueDelegate",
			ValueTypes: []evmtypes.EIP712MsgAttrType{
				{Name: "delegator_address", Type: "string"},
				{Name: "validator_address", Type: "string"},
				{Name: "amount", Type: "Coin"},
			},
			NestedTypes: []evmtypes.EIP712NestedMsgType{
				{
					Name: "Coin",
					Attrs: []evmtypes.EIP712MsgAttrType{
						{Name: "denom", Type: "string"},
						{Name: "amount", Type: "string"},
					},
				},
				{
					Name: "Vote",
					Attrs: []evmtypes.EIP712MsgAttrType{
						{Name: "voter", Type: "string"},
					},
				},
			},
		},
	}

	fromAddr := sdk.AccAddress(tests.GenerateAddress().Bytes())
	toAddr := sdk.AccAddress(tests.GenerateAddress().Bytes())
	valAddr := sdk.ValAddress(tests.GenerateAddress().Bytes())

	tests := []struct {
		name    string
		msgs    []sdk.Msg
		exp     string
		success bool
		errMsg  string
	}{
		{
			name:    "success",
			success: true,
			msgs: []sdk.Msg{
				bankTypes.NewMsgSend(fromAddr, toAddr, sdk.NewCoins(sdk.NewCoin("atom", sdkmath.NewInt(1)))),
				stakingTypes.NewMsgDelegate(fromAddr.String(), valAddr.String(), sdk.NewCoin("atom", sdkmath.NewInt(1))),
				bankTypes.NewMsgSend(fromAddr, toAddr, sdk.NewCoins(sdk.NewCoin("atom", sdkmath.NewInt(2)))),
			},
			exp: `{
				"Coin": [
					{ "name": "denom", "type": "string" },
					{ "name": "amount", "type": "string" }
				],
				"EIP712Domain": [
					{ "name": "name", "type": "string" },
					{ "name": "version", "type": "string" },
					{ "name": "chainId", "type": "uint256" },
					{ "name": "verifyingContract", "type": "string" },
					{ "name": "salt", "type": "string" }
				],
				"Fee": [
					{ "name": "amount", "type": "Coin[]" },
					{ "name": "gas", "type": "string" }
				],
				"Msg1": [
					{ "name": "type", "type": "string" },
					{ "name": "value", "type": "MsgValueSend" }
				],
				"Msg2": [
					{ "name": "type", "type": "string" },
					{ "name": "value", "type": "MsgValueDelegate" }
				],
				"Msg3": [
					{ "name": "type", "type": "string" },
					{ "name": "value", "type": "MsgValueSend" }
				],
				"MsgValueDelegate": [
					{ "name": "delegator_address", "type": "string" },
					{ "name": "validator_address", "type": "string" },
					{ "name": "amount", "type": "Coin" }
				],
				"MsgValueSend": [
					{ "name": "from_address", "type": "string" },
					{ "name": "to_address", "type": "string" },
					{ "name": "amount", "type": "Coin[]" }
				],
				"Tx": [
					{ "name": "account_number", "type": "string" },
					{ "name": "chain_id", "type": "string" },
					{ "name": "fee", "type": "Fee" },
					{ "name": "memo", "type": "string" },
					{ "name": "sequence", "type": "string" },
					{ "name": "msg1", "type": "Msg1" },
					{ "name": "msg2", "type": "Msg2" },
					{ "name": "msg3", "type": "Msg3" }
				],
				"Vote": [{ "name": "voter", "type": "string" }]
			}`,
		},
		{
			name: "fails if msg is not allowed",
			msgs: []sdk.Msg{
				bankTypes.NewMsgSend(fromAddr, toAddr, sdk.NewCoins(sdk.NewCoin("atom", sdkmath.NewInt(1)))),
				bankTypes.NewMsgMultiSend(
					bankTypes.Input{
						Address: fromAddr.String(), Coins: sdk.NewCoins(sdk.NewCoin("atom", sdkmath.NewInt(1))),
					},
					[]bankTypes.Output{
						{Address: toAddr.String(), Coins: sdk.NewCoins(sdk.NewCoin("atom", sdkmath.NewInt(1)))},
					},
				),
			},
			success: false,
			errMsg:  "eip712 message type \"/cosmos.bank.v1beta1.MsgMultiSend\" is not permitted: invalid type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgTypes, err := extractMsgTypes(tt.msgs, params)
			if tt.success {
				require.NoError(t, err)
				var expTypes apitypes.Types
				err := json.Unmarshal([]byte(tt.exp), &expTypes)
				require.NoError(t, err)
				require.Equal(t, expTypes, msgTypes)
			} else {
				require.Error(t, err)
				require.Equal(t, tt.errMsg, err.Error())
			}
		})
	}
}
