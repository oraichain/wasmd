package v050

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	icacontroller "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller"
	icacontrollerkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/keeper"
	icacontrollertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	ibcfee "github.com/cosmos/ibc-go/v8/modules/apps/29-fee"
	ibcfeekeeper "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/keeper"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/CosmWasm/wasmd/app/upgrades"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// UpgradeName defines the on-chain upgrade name
const UpgradeName = "v0.50.2"

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

		UpgradeIbcRouter(sdkCtx, ak.IBCKeeper, ak.ICAControllerKeeper, ak.IBCFeeKeeper)

		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}

func UpgradeIbcRouter(ctx sdk.Context, ibcKeeper *ibckeeper.Keeper, icaControllerKeeper icacontrollerkeeper.Keeper, ibcFeeKeeper ibcfeekeeper.Keeper) {
	ibcRouter := porttypes.NewRouter()
	ibcKeeperRoutes := ibcKeeper.Router.GetAllRoutes()

	for module, route := range ibcKeeperRoutes {
		if module == icacontrollertypes.SubModuleName {
			var icaControllerStack porttypes.IBCModule
			icaControllerStack = icacontroller.NewIBCMiddleware(icaControllerStack, icaControllerKeeper)
			icaControllerStack = ibcfee.NewIBCMiddleware(icaControllerStack, ibcFeeKeeper)

			route = icaControllerStack
		}

		ibcRouter.AddRoute(module, route)
	}

	// reset routes for ibc keeper
	ibcKeeper.ResetRouter()

	// set new routes
	ibcKeeper.SetRouter(ibcRouter)
}
