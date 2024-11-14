package v050

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
	v6 "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/migrations/v6"

	"github.com/CosmWasm/wasmd/app/upgrades"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
)

// UpgradeName defines the on-chain upgrade name
const UpgradeName = "v0.50.1"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: storetypes.StoreUpgrades{
		Added:   []string{},
		Deleted: []string{},
	},
}

func CreateUpgradeHandler(
	mm upgrades.ModuleManager,
	configurator module.Configurator,
	ak *upgrades.AppKeepers,
	keys map[string]*storetypes.KVStoreKey,
	cdc codec.BinaryCodec,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {

		sdkCtx := sdk.UnwrapSDKContext(ctx)
		if err := v6.MigrateICS27ChannelCapability(sdkCtx, cdc, keys[capabilitytypes.ModuleName], ak.CapabilityKeeper, "intertx"); err != nil {
			return nil, err
		}
		govParams, err := ak.GovKeeper.Params.Get(ctx)
		if err != nil {
			return nil, err
		}
		govParams.ExpeditedMinDeposit = govParams.MinDeposit
		err = ak.GovKeeper.Params.Set(ctx, govParams)
		if err != nil {
			return nil, err
		}

		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}
