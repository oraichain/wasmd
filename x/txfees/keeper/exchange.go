package keeper

import (
	"github.com/CosmWasm/wasmd/x/txfees/types"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) QueryOraiDexTokenExchangeRate(ctx sdk.Context, denom string) error {
	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	contractAddress := params.PriceContractAddress
	querier := wasmkeeper.Querier(k.wk)

	// build and query request
	queryData := types.BuildQueryOraidexSpotPriceRequest()
	req := &wasmtypes.QuerySmartContractStateRequest{
		Address:   contractAddress,
		QueryData: queryData,
	}

	goCtx := sdk.WrapSDKContext(ctx)
	res, err := querier.SmartContractState(goCtx, req)
	if err != nil {
		return err
	}

	// parse data and store
	rate := types.GetOraidexSpotPriceResponse(res.Data)
	err = k.SetTokenExchangeRate(ctx, denom, rate)
	if err != nil {
		return err
	}

	return nil
}
