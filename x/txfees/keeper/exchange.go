package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) QueryOraiDexTokenExchangeRate(ctx sdk.Context, denom string) error {
	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}
}
