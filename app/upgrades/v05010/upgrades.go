package v05010

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"

	"cosmossdk.io/math"
	"github.com/CosmWasm/wasmd/app/upgrades"
	"github.com/CosmWasm/wasmd/cmd/config"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
)

// UpgradeName defines the on-chain upgrade name
const UpgradeName = "v0.50.10"

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

		if err := UpgradeMintParams(sdkCtx, ak.MintKeeper); err != nil {
			return nil, err
		}

		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}

func UpgradeMintParams(ctx sdk.Context, mintKeeper *mintkeeper.Keeper) error {
	mintParams, err := mintKeeper.Params.Get(ctx)
	if err != nil {
		// in case of error, set default params
		mintParams = minttypes.DefaultParams()
		mintParams.GoalBonded = math.LegacyMustNewDecFromStr("0.67")
		mintParams.MintDenom = config.MinimalDenom
	}

	mintParams.BlocksPerYear = 45051428
	mintParams.InflationRateChange = math.LegacyMustNewDecFromStr("0.07")
	mintParams.InflationMin = mintParams.InflationRateChange
	mintParams.InflationMax = mintParams.InflationRateChange

	// set mint params
	mintKeeper.Params.Set(ctx, mintParams)

	return nil
}
