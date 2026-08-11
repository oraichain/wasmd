package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BeginBlockForks runs any hard-fork BeginForkLogic scheduled for the current height.
// The fork writes the blacklist into the txfees store; outbound-only enforcement is
// height-gated (height > ForkHeight) in RegisterBankSendRestrictions.
func BeginBlockForks(ctx sdk.Context, app *WasmApp) {
	keepers := app.GetUpgradeKeepers()
	for _, fork := range Forks {
		if ctx.BlockHeight() == fork.UpgradeHeight {
			fork.BeginForkLogic(ctx, &keepers)
			return
		}
	}
}
