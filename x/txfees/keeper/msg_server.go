package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"github.com/CosmWasm/wasmd/x/txfees/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

func (k msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if k.authority != msg.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, msg.Authority)
	}

	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}

	if err := k.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}
	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) AddFeeToken(goCtx context.Context, msg *types.MsgAddFeeToken) (*types.MsgAddFeeTokenResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if k.authority != msg.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, msg.Authority)
	}

	isAllowed, _ := k.IsTokenAllowed(ctx, msg.Config.Denom)
	if isAllowed {
		return nil, errors.Wrapf(types.ErrTokenAllowed, "fee token allowed %s", msg.Config.Denom)
	}

	k.AddAllowedToken(ctx, msg.Config.Denom)

	_, err := k.QueryOraiDexTokenExchangeRate(ctx, msg.Config.Denom)
	if err != nil {
		// we need to force this to Frozen because there is no exchange price
		msg.Config.Status = types.FeeTokenStatus_FROZEN
	} else {
		msg.Config.Status = types.FeeTokenStatus_UPDATED
	}

	err = k.SetTokenConfiguration(ctx, msg.Config)
	if err != nil {
		return nil, errors.Wrapf(types.ErrTokenAllowed, err.Error())
	}

	return &types.MsgAddFeeTokenResponse{}, nil
}

func (k msgServer) RemoveFeeToken(goCtx context.Context, msg *types.MsgRemoveFeeToken) (*types.MsgRemoveFeeTokenResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if k.authority != msg.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, msg.Authority)
	}

	isAllowed, _ := k.IsTokenAllowed(ctx, msg.Denom)
	if !isAllowed {
		return nil, errors.Wrapf(types.ErrTokenAllowed, "fee token not allowed %s", msg.Denom)
	}

	k.RemoveAllowedToken(ctx, msg.Denom)

	err := k.RemoveTokenConfiguration(ctx, msg.Denom)
	if err != nil {
		return nil, errors.Wrapf(types.ErrTokenAllowed, err.Error())
	}

	err = k.RemoveTokenExchangeRate(ctx, msg.Denom)
	if err != nil {
		return nil, errors.Wrapf(types.ErrTokenAllowed, err.Error())
	}

	return &types.MsgRemoveFeeTokenResponse{}, nil
}
