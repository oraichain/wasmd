package keeper

import (
	"cosmossdk.io/math"
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

// Fee token configuration store
func (k Keeper) SetTokenConfiguration(ctx sdk.Context, denom string, config types.FeeTokenConfiguration) error {
	store := k.storeService.OpenKVStore(ctx)
	key := types.GetTokenConfigurationKey(denom)

	bz, err := k.cdc.Marshal(&config)
	if err != nil {
		return err
	}

	store.Set(key, bz)
	return nil
}

func (k Keeper) GetTokenConfiguration(ctx sdk.Context, denom string) (types.FeeTokenConfiguration, bool) {
	store := k.storeService.OpenKVStore(ctx)
	key := types.GetTokenConfigurationKey(denom)

	bz, _ := store.Get(key)
	if bz == nil {
		return types.FeeTokenConfiguration{}, false
	}

	var config types.FeeTokenConfiguration
	k.cdc.MustUnmarshal(bz, &config)

	return config, true
}

func (k Keeper) RemoveTokenConfiguration(ctx sdk.Context, denom string) error {
	store := k.storeService.OpenKVStore(ctx)
	key := types.GetTokenConfigurationKey(denom)

	return store.Delete(key)
}

// Token exchange rate store
func (k Keeper) SetTokenExchangeRate(ctx sdk.Context, denom string, price math.LegacyDec) error {
	store := k.storeService.OpenKVStore(ctx)
	key := types.GetTokenExchangeRateKey(denom)
	bz, err := price.Marshal()
	if err != nil {
		return err
	}

	store.Set(key, bz)
	return nil
}

func (k Keeper) GetTokenExchangeRate(ctx sdk.Context, denom string) (math.LegacyDec, bool) {
	store := k.storeService.OpenKVStore(ctx)
	key := types.GetTokenExchangeRateKey(denom)

	bz, _ := store.Get(key)
	if bz == nil {
		return math.LegacyDec{}, false
	}

	var rate math.LegacyDec
	if err := rate.Unmarshal(bz); err != nil {
		return math.LegacyDec{}, false
	}

	return rate, true
}

func (k Keeper) RemoveTokenExchangeRate(ctx sdk.Context, denom string) error {
	store := k.storeService.OpenKVStore(ctx)
	key := types.GetTokenExchangeRateKey(denom)
	return store.Delete(key)
}

// base denom
func (k Keeper) SetBaseTokenDenom(ctx sdk.Context, denom string) error {
	store := k.storeService.OpenKVStore(ctx)
	return store.Set(types.BaseDenomKey, []byte(denom))
}

func (k Keeper) GetBaseTokenDenom(ctx sdk.Context) (denom string, err error) {
	store := k.storeService.OpenKVStore(ctx)

	has, err := store.Has(types.BaseDenomKey)
	if !has || err != nil {
		return "", err
	}

	bz, err := store.Get(types.BaseDenomKey)
	if err != nil {
		return "", err
	}

	return string(bz), nil
}
