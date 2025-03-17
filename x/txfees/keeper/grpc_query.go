package keeper

import (
	"context"

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
