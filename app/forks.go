package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/app/upgrades"
)

// BeginBlockForks runs any hard-fork BeginForkLogic scheduled for the current height.
func BeginBlockForks(ctx sdk.Context, app *WasmApp) {
	keepers := app.GetUpgradeKeepers()
	for _, fork := range Forks {
		if ctx.BlockHeight() == fork.UpgradeHeight {
			fork.BeginForkLogic(ctx, &keepers)
			return
		}
	}
}

// GetUpgradeKeepers returns the keepers bundle used by upgrade/fork handlers.
func (app *WasmApp) GetUpgradeKeepers() upgrades.AppKeepers {
	return upgrades.AppKeepers{
		AccountKeeper:             &app.AccountKeeper,
		ParamsKeeper:              &app.ParamsKeeper,
		ConsensusParamsKeeper:     &app.ConsensusParamsKeeper,
		CapabilityKeeper:          app.CapabilityKeeper,
		ScopedICAControllerKeeper: &app.ScopedICAControllerKeeper,
		ScopedIBCKeeper:           &app.ScopedIBCKeeper,
		IBCKeeper:                 app.IBCKeeper,
		MintKeeper:                &app.MintKeeper,
		GovKeeper:                 &app.GovKeeper,
		TxFeesKeeper:              app.TxFeesKeeper,
		ICAControllerKeeper:       app.ICAControllerKeeper,
		IBCFeeKeeper:              app.IBCFeeKeeper,
		EvmKeeper:                 nil,
		Codec:                     app.appCodec,
		GetStoreKey:               app.GetKey,
	}
}
