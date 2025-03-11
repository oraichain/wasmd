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

	config, found := k.GetTokenConfiguration(ctx, inputFee.Denom)
	if !found {
		return sdk.Coin{}, types.ErrTokenConfigurationNotFound
	}

	if config.Status != types.FeeTokenStatus_UPDATED {
		return sdk.Coin{}, types.ErrFeeTokenUnAvailable
	}

	rate, found := k.GetTokenExchangeRate(ctx, inputFee.Denom)
	if !found {
		return sdk.Coin{}, types.ErrInvalidExchangeRate
	}

	baseAmount := rate.MulInt(inputFee.Amount).RoundInt()

	return sdk.NewCoin(baseDenom, baseAmount), nil
}
