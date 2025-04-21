package json

import (
	gjson "encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/evm/x/vm/core/vm"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

func (p Precompile) ExtractAsBytes(ctx sdk.Context, contract *vm.Contract, method *abi.Method, args []interface{}) ([]byte, error) {
	result, err := parseExtractAsBytesArgs(args)
	if err != nil {
		return nil, err
	}

	// in the case of a string value, remove the quotes
	if len(result) >= 2 && result[0] == '"' && result[len(result)-1] == '"' {
		result = result[1 : len(result)-1]
	}

	ret, rerr := method.Outputs.Pack([]byte(result))
	if rerr != nil {
		return nil, rerr
	}

	return ret, nil
}

func (p Precompile) ExtractAsBytesList(ctx sdk.Context, contract *vm.Contract, method *abi.Method, args []interface{}) ([]byte, error) {
	result, err := parseExtractAsBytesArgs(args)
	if err != nil {
		return nil, err
	}

	decodedResult := []gjson.RawMessage{}
	if err := gjson.Unmarshal(result, &decodedResult); err != nil {
		return nil, err
	}

	decodedBytes := [][]byte{}
	for _, r := range decodedResult {
		decodedBytes = append(decodedBytes, []byte(r))
	}

	ret, rerr := method.Outputs.Pack(decodedBytes)
	if rerr != nil {
		return nil, rerr
	}

	return ret, nil
}

func (p Precompile) ExtractAsUint256(ctx sdk.Context, contract *vm.Contract, method *abi.Method, args []interface{}) ([]byte, error) {

	byteArr := make([]byte, 32)
	uint_, err := parseExtractAsUint256Args(args)
	if err != nil {
		return nil, err
	}

	if uint_.BitLen() > 256 {
		return nil, fmt.Errorf("value does not fit in 32 bytes\n")
	}

	uint_.FillBytes(byteArr)

	return byteArr, nil
}
