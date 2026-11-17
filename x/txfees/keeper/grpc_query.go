package keeper

import (
	"context"

	errors "cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"
	"github.com/CosmWasm/wasmd/x/txfees/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func (k Keeper) CheckBlacklist(ctx context.Context, req *types.QueryCheckBlacklistRequest) (*types.QueryCheckBlacklistResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	addr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid address %s", req.Address)
	}

	blacklisted, err := k.IsBlacklisted(sdkCtx, addr)
	if err != nil {
		return nil, err
	}

	return &types.QueryCheckBlacklistResponse{Blacklisted: blacklisted}, nil
}

// BlacklistAddresses returns a paginated page of the blacklist. This query is
// exposed over gRPC and REST with no gas meter bounding it, so it must never
// iterate the whole prefix into one unbounded response.
func (k Keeper) BlacklistAddresses(ctx context.Context, req *types.QueryBlacklistAddressesRequest) (*types.QueryBlacklistAddressesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	blacklistStore := prefix.NewStore(
		runtime.KVStoreAdapter(k.storeService.OpenKVStore(sdkCtx)),
		types.BlacklistKeyPrefix,
	)

	var addresses []string
	pageRes, err := query.Paginate(blacklistStore, req.Pagination, func(_, value []byte) error {
		addresses = append(addresses, sdk.AccAddress(value).String())
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryBlacklistAddressesResponse{Addresses: addresses, Pagination: pageRes}, nil
}
