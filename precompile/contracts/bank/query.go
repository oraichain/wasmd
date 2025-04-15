package bank

import (
	"fmt"

	pcommon "github.com/CosmWasm/wasmd/precompile/common"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/evm/x/vm/core/vm"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

func (p Precompile) Balance(ctx sdk.Context, contract *vm.Contract, method *abi.Method, args []interface{}) ([]byte, error) {
	if err := pcommon.ValidateArgsLength(args, 2); err != nil {
		return nil, err
	}
	evmAddr := args[0].(common.Address)
	cosmosAddr := p.evmKeeper.GetCosmosAddressMapping(ctx, evmAddr)
	denom := args[1].(string)

	balance := p.bankKeeper.GetBalance(ctx, cosmosAddr, denom)

	ret, rerr := method.Outputs.Pack(balance.Amount.BigInt())
	if rerr != nil {
		return nil, rerr
	}

	return ret, nil
}

func (p Precompile) AllBalances(ctx sdk.Context, contract *vm.Contract, method *abi.Method, args []interface{}) ([]byte, error) {
	if err := pcommon.ValidateArgsLength(args, 1); err != nil {
		return nil, err
	}
	evmAddr := args[0].(common.Address)
	cosmosAddr := p.evmKeeper.GetCosmosAddressMapping(ctx, evmAddr)
	coins := p.bankKeeper.GetAllBalances(ctx, cosmosAddr)
	coinBalances := make([]CoinBalance, 0, len(coins))
	for _, coin := range coins {
		coinBalances = append(coinBalances, CoinBalance{
			Amount: coin.Amount.BigInt(),
			Denom:  coin.Denom,
		})
	}

	ret, rerr := method.Outputs.Pack(coinBalances)
	if rerr != nil {
		return nil, rerr
	}

	return ret, nil
}

func (p Precompile) Name(ctx sdk.Context, contract *vm.Contract, method *abi.Method, args []interface{}) ([]byte, error) {
	if err := pcommon.ValidateArgsLength(args, 1); err != nil {
		return nil, err
	}
	denom := args[0].(string)
	metadata, found := p.bankKeeper.GetDenomMetaData(ctx, denom)
	if !found {
		return nil, fmt.Errorf("could not find the metadata of denom %s", denom)
	}

	ret, rerr := method.Outputs.Pack(metadata.Name)
	if rerr != nil {
		return nil, rerr
	}

	return ret, nil
}

func (p Precompile) Symbol(ctx sdk.Context, contract *vm.Contract, method *abi.Method, args []interface{}) ([]byte, error) {
	if err := pcommon.ValidateArgsLength(args, 1); err != nil {
		return nil, err
	}
	denom := args[0].(string)
	metadata, found := p.bankKeeper.GetDenomMetaData(ctx, denom)
	if !found {
		return nil, fmt.Errorf("could not find the metadata of denom %s", denom)
	}

	ret, rerr := method.Outputs.Pack(metadata.Symbol)
	if rerr != nil {
		return nil, rerr
	}

	return ret, nil
}

func (p Precompile) Decimals(ctx sdk.Context, contract *vm.Contract, method *abi.Method, args []interface{}) ([]byte, error) {
	if err := pcommon.ValidateArgsLength(args, 1); err != nil {
		return nil, err
	}

	ret, rerr := method.Outputs.Pack(uint8(0))
	if rerr != nil {
		return nil, rerr
	}

	return ret, nil
}

func (p Precompile) Supply(ctx sdk.Context, contract *vm.Contract, method *abi.Method, args []interface{}) ([]byte, error) {
	if err := pcommon.ValidateArgsLength(args, 1); err != nil {
		return nil, err
	}
	denom := args[0].(string)
	supply := p.bankKeeper.GetSupply(ctx, denom)

	ret, rerr := method.Outputs.Pack(supply.Amount.BigInt())
	if rerr != nil {
		return nil, rerr
	}

	return ret, nil
}
