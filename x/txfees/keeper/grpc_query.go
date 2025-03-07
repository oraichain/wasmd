package keeper

import (
	"context"

	"github.com/CosmWasm/wasmd/x/txfees/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) Params(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	_ = sdk.UnwrapSDKContext(ctx)
	// params := k.GetParams(sdkCtx)

	return &types.QueryParamsResponse{}, nil
}
