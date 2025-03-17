package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/CosmWasm/wasmd/x/txfees/types"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) QueryOraiDexTokenExchangeRate(ctx sdk.Context, denom string) (math.LegacyDec, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return math.LegacyDec{}, err
	}

	contractAddress := params.PriceContractAddress
	querier := wasmkeeper.Querier(k.wk)

	// build and query request
	queryData, err := types.BuildQueryOraidexSpotPriceRequest(denom)
	if err != nil {
		return math.LegacyDec{}, err
	}

	req := &wasmtypes.QuerySmartContractStateRequest{
		Address:   contractAddress,
		QueryData: queryData,
	}

	goCtx := sdk.WrapSDKContext(ctx)
	res, err := querier.SmartContractState(goCtx, req)
	if err != nil {
		return math.LegacyDec{}, err
	}

	// parse data and store
	sqrtPrice, err := types.GetOraidexSqrtPriceResponse(res.Data)
	if err != nil {
		return math.LegacyDec{}, err
	}

	exchangeRate := sqrtPrice.Power(2)
	k.Logger(ctx).Info(fmt.Sprintf("set token exchange rate: token %s rate %s", denom, exchangeRate.String()))

	return exchangeRate, nil
}

func (k Keeper) ConvertToBaseTokenFee(ctx sdk.Context, inputFee sdk.Coin) (sdk.Coin, error) {
	baseDenom, err := k.GetBaseTokenDenom(ctx)
	if err != nil {
		return sdk.Coin{}, err
	}

	if inputFee.Denom == baseDenom {
		return inputFee, nil
	}

	isAllowed, err := k.IsTokenAllowed(ctx, inputFee.Denom)
	if err != nil {
		return sdk.Coin{}, err
	}

	if !isAllowed {
		return sdk.Coin{}, types.ErrTokenAllowed
	}

	rate, err := k.QueryOraiDexTokenExchangeRate(ctx, inputFee.Denom)
	if err != nil {
		return sdk.Coin{}, err
	}

	// rate = quote_asset/base_asset
	// => base_amount = rate * quote_asset_amount
	baseAmount := rate.MulInt(inputFee.Amount).RoundInt()

	return sdk.NewCoin(baseDenom, baseAmount), nil
}
