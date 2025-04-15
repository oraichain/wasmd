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
	evmAddress, err := ParseGetCosmosAddrArgs(args)
	if err != nil {
		return nil, err
	}

	cosmosAddress := p.EVMKeeper.GetCosmosAddressMapping(ctx, evmAddress)

	return method.Outputs.Pack(cosmosAddress)
}

func (p Precompile) GetEvmAddr(
	ctx sdk.Context,
	contract *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	cosmosAddress, err := ParseGetEvmAddrArgs(args)
	if err != nil {
		return nil, err
	}

	cosmosSdkAddress, err := sdk.AccAddressFromBech32(cosmosAddress)
	if err != nil {
		return nil, err
	}

	evmAddress, err := p.EVMKeeper.GetEvmAddressMapping(ctx, cosmosSdkAddress)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(evmAddress)
}
