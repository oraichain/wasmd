package keeper

import (
	"fmt"

	"cosmossdk.io/math"
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

// Fee token configuration store
func (k Keeper) SetTokenConfiguration(ctx sdk.Context, denom string, config types.FeeTokenConfiguration) error {
	store := ctx.KVStore(k.storeKey)
	key := types.GetTokenConfigurationKey(denom)

	bz, err := k.cdc.Marshal(&config)
	if err != nil {
		return err
	}

	store.Set(key, bz)
	return nil
}

func (k Keeper) GetTokenConfiguration(ctx sdk.Context, denom string) (types.FeeTokenConfiguration, bool) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetTokenConfigurationKey(denom)

	bz := store.Get(key)
	if bz == nil {
		return types.FeeTokenConfiguration{}, false
	}

	var config types.FeeTokenConfiguration
	k.cdc.MustUnmarshal(bz, &config)

	return config, true
}

// Token exchange rate store
func (k Keeper) SetTokenExchangeRate(ctx sdk.Context, denom string, price math.LegacyDec) error {
	store := ctx.KVStore(k.storeKey)
	key := types.GetTokenExchangeRateKey(denom)
	bz, err := price.Marshal()
	if err != nil {
		return err
	}

	store.Set(key, bz)
	return nil
}

func (k Keeper) GetTokenExchangeRate(ctx sdk.Context, denom string) (math.LegacyDec, bool) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetTokenExchangeRateKey(denom)

	bz := store.Get(key)
	if bz == nil {
		return math.LegacyDec{}, false
	}

	var rate math.LegacyDec
	if err := rate.Unmarshal(bz); err != nil {
		return math.LegacyDec{}, false
	}

	return rate, true
}

func (k Keeper) RemoveTokenExchangeRate(ctx sdk.Context, denom string) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetTokenExchangeRateKey(denom)
	store.Delete(key)
}
