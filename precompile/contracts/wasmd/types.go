package wasmd

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	cmn "github.com/cosmos/evm/precompiles/common"
)

/*
ParseInstantiateArgs parses the call arguments for the wasmd Instantiate contracts

	codeID := res[0].(uint64)
	admin := res[1].(string)
	msg := res[2].([]byte)
	label := res[3].(string)
	funds := res[4].([]byte)
*/
func ParseInstantiateArgs(args []interface{}) (uint64, sdk.AccAddress, []byte, string, sdk.Coins, error) {
	if len(args) != 5 {
		return 0, sdk.AccAddress{}, []byte{}, "", sdk.Coins{}, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 5, len(args))
	}

	codeID, ok := args[0].(uint64)
	if !ok {
		return 0, sdk.AccAddress{}, []byte{}, "", sdk.Coins{}, fmt.Errorf("invalid code ID: %v", args[0])
	}

	admin, ok := args[1].(string)
	if !ok {
		return 0, sdk.AccAddress{}, []byte{}, "", sdk.Coins{}, fmt.Errorf("invalid admin: %v", args[1])
	}
	adminAddr, err := sdk.AccAddressFromBech32(admin)
	if err != nil {
		return 0, sdk.AccAddress{}, []byte{}, "", sdk.Coins{}, fmt.Errorf("invalid admin: %v", args[1])
	}

	msg, ok := args[2].([]byte)
	if !ok {
		return 0, sdk.AccAddress{}, []byte{}, "", sdk.Coins{}, fmt.Errorf("invalid msg: %v", args[2])
	}

	label, ok := args[3].(string)
	if !ok {
		return 0, sdk.AccAddress{}, []byte{}, "", sdk.Coins{}, fmt.Errorf("invalid label: %v", args[3])
	}

	fundsBz, ok := args[4].([]byte)
	if !ok {
		return 0, sdk.AccAddress{}, []byte{}, "", sdk.Coins{}, fmt.Errorf("invalid funds: %v", args[4])
	}
	funds := UnmarshalCosmWasmDeposit(fundsBz)

	return codeID, adminAddr, msg, label, funds, nil
}

/*
ParseExecuteArgs parses the call arguments for the wasmd execute contracts

	contractAddress := res[0].(string)
	msg := res[1].([]byte)
	funds := res[2].([]byte)
*/
func ParseExecuteArgs(args []interface{}) (sdk.AccAddress, []byte, sdk.Coins, error) {
	if len(args) != 3 {
		return sdk.AccAddress{}, []byte{}, sdk.Coins{}, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 3, len(args))
	}

	contract, ok := args[0].(string)
	if !ok {
		return sdk.AccAddress{}, []byte{}, sdk.Coins{}, fmt.Errorf("invalid contract: %v", args[0])
	}
	contractAddress, err := sdk.AccAddressFromBech32(contract)
	if err != nil {
		return sdk.AccAddress{}, []byte{}, sdk.Coins{}, fmt.Errorf("invalid contract address: %v", args[1])
	}

	msg, ok := args[1].([]byte)
	if !ok {
		return sdk.AccAddress{}, []byte{}, sdk.Coins{}, fmt.Errorf("invalid msg: %v", args[2])
	}

	fundsBz, ok := args[2].([]byte)
	if !ok {
		return sdk.AccAddress{}, []byte{}, sdk.Coins{}, fmt.Errorf("invalid funds: %v", args[4])
	}
	funds := UnmarshalCosmWasmDeposit(fundsBz)

	return contractAddress, msg, funds, nil
}

/*
ParseQueryArgs parses the call arguments for the wasmd query contracts

	contractAddress := res[0].(string)
	req := res[1].([]byte)
*/
func ParseQueryArgs(args []interface{}) (sdk.AccAddress, []byte, error) {
	if len(args) != 3 {
		return sdk.AccAddress{}, []byte{}, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 3, len(args))
	}

	contract, ok := args[0].(string)
	if !ok {
		return sdk.AccAddress{}, []byte{}, fmt.Errorf("invalid contract: %v", args[0])
	}
	contractAddress, err := sdk.AccAddressFromBech32(contract)
	if err != nil {
		return sdk.AccAddress{}, []byte{}, fmt.Errorf("invalid contract address: %v", args[1])
	}

	req, ok := args[1].([]byte)
	if !ok {
		return sdk.AccAddress{}, []byte{}, fmt.Errorf("invalid request: %v", args[2])
	}

	return contractAddress, req, nil
}

func UnmarshalCosmWasmDeposit(coins []byte) sdk.Coins {
	// unmarshal coins
	var deposit sdk.Coins
	err := json.Unmarshal(coins, &deposit)
	if err != nil {
		return sdk.NewCoins()
	}
	return deposit
}
