package v050

import (
	"context"

	// govv1beta1 "cosmossdk.io/api/cosmos/gov/v1beta1"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	// govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
	v6 "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/migrations/v6"

	"github.com/CosmWasm/wasmd/app/upgrades"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	channelkeeper "github.com/cosmos/ibc-go/v8/modules/core/04-channel/keeper"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
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

		if err := ReleaseWrongIcaControllerCaps(sdkCtx, ak.IBCKeeper.ChannelKeeper, ak.ScopedICAControllerKeeper); err != nil {
			return nil, err
		}

		if err := upgradeGovParams(sdkCtx, ak.ParamsKeeper); err != nil {
			return nil, err
		}

		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}

func ReleaseWrongIcaControllerCaps(ctx sdk.Context, channelKeeper channelkeeper.Keeper, scopedICAControllerKeeper *capabilitykeeper.ScopedKeeper) error {
	chanels := channelKeeper.GetAllChannelsWithPortPrefix(ctx, icatypes.ControllerPortPrefix)
	for _, ch := range chanels {
		name := host.ChannelCapabilityPath(ch.PortId, ch.ChannelId)
		cap, found := scopedICAControllerKeeper.GetCapability(ctx, name)
		if found {
			if err := scopedICAControllerKeeper.ReleaseCapability(ctx, cap); err != nil {
				return err
			}
		}
	}

	return nil
}

func upgradeGovParams(ctx sdk.Context, paramsKeeper *paramskeeper.Keeper) error {
	// govSubspace, exist := paramsKeeper.GetSubspace(govtypes.ModuleName)
	// if !exist {
	// 	return errors.New("gov params space must existed")
	// }

	// var govParams govv1beta1.VotingParams
	// govSubspace.GetParamSet(ctx, &govParams)
	// if mintParams.InflationMin.GT(mintParams.InflationMax) {
	// 	mintParams.InflationMin = mintParams.InflationMax
	// 	mintSpace.SetParamSet(ctx, &mintParams)
	// }

	return nil
}
