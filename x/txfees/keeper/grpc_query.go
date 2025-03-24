package keeper

import (
	"context"

	errors "cosmossdk.io/errors"
	"github.com/CosmWasm/wasmd/x/txfees/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) Params(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params, _ := k.GetParams(sdkCtx)

	return &types.QueryParamsResponse{
		Params: params,
	}, nil
}

func (k Keeper) AllowedTokens(ctx context.Context, _ *types.QueryAllowedTokensRequest) (*types.QueryAllowedTokensResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	var tokens []string

	k.IterateAllowedTokenList(sdkCtx, func(denom string) (stop bool) {
		tokens = append(tokens, denom)

		return false
	})

	return &types.QueryAllowedTokensResponse{Tokens: tokens}, nil
}

func (k Keeper) TokenExchangeRate(ctx context.Context, req *types.QueryTokenExchangeRateRequest) (*types.QueryTokenExchangeRateResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	allowed, err := k.IsTokenAllowed(sdkCtx, req.Denom)
	if err != nil {
		return nil, err
	}

	if !allowed {
		return nil, errors.Wrapf(types.ErrTokenAllowed, "token %s is not allowed", req.Denom)
	}

	baseDenom, err := k.GetBaseTokenDenom(sdkCtx)
	if err != nil {
		return nil, err
	}

	rate, err := k.QueryOraiDexTokenExchangeRate(sdkCtx, req.Denom)
	if err != nil {
		return nil, err
	}

	return &types.QueryTokenExchangeRateResponse{
		BaseDenom:  baseDenom,
		QuoteDenom: req.Denom,
		Rate:       rate,
	}, nil
}
