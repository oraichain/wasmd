package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BeginBlockForks runs any hard-fork BeginForkLogic scheduled for the current height.
// Blacklist set is activated at app startup; outbound-only enforcement is height-gated
// (height > ForkHeight) in BlacklistSendRestriction.
func BeginBlockForks(ctx sdk.Context, app *WasmApp) {
	keepers := app.GetUpgradeKeepers()
	for _, fork := range Forks {
		if ctx.BlockHeight() == fork.UpgradeHeight {
			fork.BeginForkLogic(ctx, &keepers)
			return
		}
	}
}
