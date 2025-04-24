package wasmd

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/evm/x/vm/core/vm"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

func (p Precompile) Instantiate(
	ctx sdk.Context,
	contract *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	from := contract.CallerAddress
	codeID, admin, msg, label, funds, err := ParseInstantiateArgs(args)
	if err != nil {
		return nil, err
	}

	return p.instantiateCosmWasm(ctx, method, from, codeID, admin, msg, label, funds)
}

func (p Precompile) Execute(
	ctx sdk.Context,
	contract *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	from := contract.CallerAddress
	contractAddress, msg, funds, err := ParseExecuteArgs(args)
	if err != nil {
		return nil, err
	}

	return p.executeCosmWasm(ctx, method, from, contractAddress, msg, funds)
}

func (p Precompile) Query(
	ctx sdk.Context,
	contract *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	contractAddress, req, err := ParseQueryArgs(args)
	if err != nil {
		return nil, err
	}

	return p.queryCosmWasm(ctx, method, contractAddress, req)
}
