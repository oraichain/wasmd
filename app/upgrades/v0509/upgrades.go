package v0509

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/CosmWasm/wasmd/app/upgrades"
	precisebanktypes "github.com/CosmWasm/wasmd/x/precisebank/types"
	txfeestypes "github.com/CosmWasm/wasmd/x/txfees/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const (
	USDAI = "factory/orai1wuvhex9xqs3r539mvc6mtm7n20fcj3qr2m0y9khx6n5vtlngfzes3k0rq9/DYeTA4ZQhEwoJ5imjq1Q3zgwfTgkh4WmdfFHAq3jLrv3"
)

// UpgradeName defines the on-chain upgrade name
const UpgradeName = "v0.50.9"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: storetypes.StoreUpgrades{
		Added:   []string{precisebanktypes.StoreKey, txfeestypes.StoreKey},
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

		// set params
		ak.TxFeesKeeper.SetParams(sdkCtx, txfeestypes.DefaultParams())
		ak.TxFeesKeeper.AddAllowedToken(sdkCtx, USDAI)

		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}
