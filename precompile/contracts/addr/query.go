package addr

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/evm/x/vm/core/vm"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

func (p Precompile) GetCosmosAddr(
	ctx sdk.Context,
	contract *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	// TODO: implement the logic for the getCosmosAddr method

	return nil, nil
}

func (p Precompile) GetEvmAddr(
	ctx sdk.Context,
	contract *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	// TODO: implement the logic for the getEvmAddr method

	return nil, nil
}
