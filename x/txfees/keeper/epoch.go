package keeper

import (
	"fmt"
	"time"

	storetypes "cosmossdk.io/store/types"
	"github.com/CosmWasm/wasmd/x/txfees/types"
	"github.com/cosmos/cosmos-sdk/runtime"
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

func (k Keeper) SetEpochInfo(ctx sdk.Context, epoch types.EpochInfo) error {
	store := k.storeService.OpenKVStore(ctx)
	key := types.GetEpochKey(epoch.Identifier)

	bz, err := k.cdc.Marshal(&epoch)
	if err != nil {
		return err
	}

	return store.Set(key, bz)
}

func (k Keeper) GetEpochInfo(ctx sdk.Context, identifier string) (types.EpochInfo, bool) {
	store := k.storeService.OpenKVStore(ctx)
	key := types.GetEpochKey(identifier)

	bz, err := store.Get(key)
	if bz == nil || err != nil {
		return types.EpochInfo{}, false
	}

	epoch := types.EpochInfo{}
	err = k.cdc.Unmarshal(bz, &epoch)
	if err != nil {
		panic(err)
	}

	return epoch, true
}

func (k Keeper) AddEpochInfo(ctx sdk.Context, epoch types.EpochInfo) error {
	err := epoch.Validate()
	if err != nil {
		return err
	}
	// Check if identifier already exists
	if has, _ := k.HasEpochInfo(ctx, epoch.Identifier); !has {
		return fmt.Errorf("epoch with identifier %s already exists", epoch.Identifier)
	}

	// Initialize empty and default epoch values
	if epoch.StartTime.Equal(time.Time{}) {
		epoch.StartTime = ctx.BlockTime()
	}
	epoch.CurrentEpochStartHeight = ctx.BlockHeight()
	return k.SetEpochInfo(ctx, epoch)
}

func (k Keeper) IterateEpochInfo(ctx sdk.Context, cb func(index int64, epochInfo types.EpochInfo) (stop bool)) {
	store := k.storeService.OpenKVStore(ctx)

	iterator := storetypes.KVStorePrefixIterator(runtime.KVStoreAdapter(store), types.EpochKeyPrefix)
	defer iterator.Close()

	i := int64(0)

	for ; iterator.Valid(); iterator.Next() {
		epoch := types.EpochInfo{}
		err := k.cdc.Unmarshal(iterator.Value(), &epoch)
		if err != nil {
			panic(err)
		}

		stop := cb(i, epoch)
		if stop {
			break
		}
		i++
	}
}

func (k Keeper) AllEpochInfos(ctx sdk.Context) []types.EpochInfo {
	epochs := []types.EpochInfo{}
	k.IterateEpochInfo(ctx, func(index int64, epoch types.EpochInfo) (stop bool) {
		epochs = append(epochs, epoch)
		return false
	})

	return epochs
}

func (k Keeper) AfterEpochEnd(ctx sdk.Context, identifier string) {
	switch identifier {
	case types.DefaultQueryEpochIdentifier:
		k.IterateAllowedTokenList(ctx, func(denom string) (stop bool) {
			k.QueryOraiDexTokenExchangeRate(ctx, denom)
			return false
		})
	default:
		k.Logger(ctx).Error((fmt.Sprintf("unknown epoch %s", identifier)))
	}
}
