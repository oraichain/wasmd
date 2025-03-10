package keeper

import (
	"github.com/CosmWasm/wasmd/x/txfees/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) HasEpochInfo(ctx sdk.Context, identifier string) (bool, error) {
	store := k.storeService.OpenKVStore(ctx)
	key := types.GetEpochKey(identifier)

	has, err := store.Has(key)
	if err != nil {
		return false, err
	}

	return has, nil
}
