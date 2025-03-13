package keeper

import (
	"context"

	"github.com/CosmWasm/wasmd/x/txfees/types"

	errors "cosmossdk.io/errors"
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

func (k Keeper) TokensConfig(ctx context.Context, _ *types.QueryTokensConfigRequest) (*types.QueryTokensConfigResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	var configs []types.FeeTokenConfiguration
	var denoms []string

	k.IterateAllowedTokenList(sdkCtx, func(denom string) (stop bool) {
		denoms = append(denoms, denom)

		return false
	})

	for _, denom := range denoms {
		config, found := k.GetTokenConfiguration(sdkCtx, denom)
		if !found {
			return &types.QueryTokensConfigResponse{}, errors.Wrapf(types.ErrTokenConfigurationNotFound, "token configuration not found %s", denom)
		}

		configs = append(configs, config)
	}

	return &types.QueryTokensConfigResponse{Configs: configs}, nil
}
