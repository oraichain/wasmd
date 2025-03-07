package keeper

import (
	"fmt"

	storetypes "cosmossdk.io/store/types"
	"github.com/CosmWasm/wasmd/x/txfees/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Allowed token store
func (k Keeper) AddAllowedToken(ctx sdk.Context, denom string) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetAllowedTokenKey(denom)
	store.Set(key, []byte{1})
}

func (k Keeper) RemoveAllowedToken(ctx sdk.Context, denom string) error {
	store := ctx.KVStore(k.storeKey)
	key := types.GetAllowedTokenKey(denom)

	if !store.Has(key) {
		return fmt.Errorf("denom not allowed %s", denom)
	}
	store.Delete(key)
	return nil
}

func (k Keeper) IsTokenAllowed(ctx sdk.Context, denom string) bool {
	store := ctx.KVStore(k.storeKey)
	key := types.GetAllowedTokenKey(denom)
	return store.Has(key)
}

func (k Keeper) IterateAllowedTokenList(ctx sdk.Context, cb func(denom string) (stop bool)) {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, types.AllowedTokenKeyPrefix)

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		denom := string(iterator.Key())
		if cb(denom) {
			break
		}
	}
}
