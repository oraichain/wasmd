package keeper

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/CosmWasm/wasmd/x/txfees/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Allowed token store
func (k Keeper) AddAllowedToken(ctx sdk.Context, denom string) {
	store := k.storeService.OpenKVStore(ctx)
	key := types.GetAllowedTokenKey(denom)
	store.Set(key, []byte(denom))
}

func (k Keeper) RemoveAllowedToken(ctx sdk.Context, denom string) error {
	store := k.storeService.OpenKVStore(ctx)
	key := types.GetAllowedTokenKey(denom)
	store.Delete(key)
	return nil
}

func (k Keeper) IsTokenAllowed(ctx sdk.Context, denom string) (bool, error) {
	store := k.storeService.OpenKVStore(ctx)
	key := types.GetAllowedTokenKey(denom)
	has, err := store.Has(key)
	if err != nil {
		return false, err
	}
	return has, nil
}

func (k Keeper) IterateAllowedTokenList(ctx sdk.Context, cb func(denom string) (stop bool)) {
	store := k.storeService.OpenKVStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(runtime.KVStoreAdapter(store), types.AllowedTokenKeyPrefix)

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		denom := string(iterator.Value())
		if cb(denom) {
			break
		}
	}
}

// Blacklist store
func (k Keeper) AddBlacklist(ctx sdk.Context, address sdk.AccAddress) {
	store := k.storeService.OpenKVStore(ctx)
	key := types.GetBlacklistKey(address)
	store.Set(key, []byte(address))
}

func (k Keeper) IsBlacklisted(ctx sdk.Context, address sdk.AccAddress) (bool, error) {
	store := k.storeService.OpenKVStore(ctx)
	key := types.GetBlacklistKey(address)
	has, err := store.Has(key)
	if err != nil {
		return false, err
	}
	return has, nil
}
