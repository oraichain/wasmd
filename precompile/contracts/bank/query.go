package bank

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/evm/x/vm/core/vm"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

func (p Precompile) Balance(ctx sdk.Context, contract *vm.Contract, method *abi.Method, args []interface{}) ([]byte, error) {
	evmAddr, denom, err := parseBalanceArgs(args)
	if err != nil {
		return nil, err
	}
	cosmosAddr := p.evmKeeper.GetCosmosAddressMapping(ctx, evmAddr)

	balance := p.bankKeeper.GetBalance(ctx, cosmosAddr, denom)

	ret, rerr := method.Outputs.Pack(balance.Amount.BigInt())
	if rerr != nil {
		return nil, rerr
	}

	return ret, nil
}

func (p Precompile) AllBalances(ctx sdk.Context, contract *vm.Contract, method *abi.Method, args []interface{}) ([]byte, error) {
	evmAddr, err := parseAllBalancesArgs(args)
	if err != nil {
		return nil, err
	}
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
	denom, err := parseErc20Args(args)
	if err != nil {
		return nil, err
	}
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
	denom, err := parseErc20Args(args)
	if err != nil {
		return nil, err
	}
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
	denom, err := parseErc20Args(args)
	if err != nil {
		return nil, err
	}
	_, found := p.bankKeeper.GetDenomMetaData(ctx, denom)
	if !found {
		return nil, fmt.Errorf("could not find the metadata of denom %s", denom)
	}

	ret, rerr := method.Outputs.Pack(uint8(0))
	if rerr != nil {
		return nil, rerr
	}

	return ret, nil
}

func (p Precompile) Supply(ctx sdk.Context, contract *vm.Contract, method *abi.Method, args []interface{}) ([]byte, error) {
	denom, err := parseErc20Args(args)
	if err != nil {
		return nil, err
	}
	supply := p.bankKeeper.GetSupply(ctx, denom)

	ret, rerr := method.Outputs.Pack(supply.Amount.BigInt())
	if rerr != nil {
		return nil, rerr
	}

	return ret, nil
}
