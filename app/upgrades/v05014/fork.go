package v10

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/app/upgrades"
)

// RunForkLogic runs one-time hard-fork state transitions at ForkHeight.
// TODO: implement production fork logic, then re-run scripts/localfork verification.
func RunForkLogic(ctx sdk.Context, appKeepers *upgrades.AppKeepers) {
	ctx.Logger().Info("========== running v0.50.14 fork logic ==========", "height", ctx.BlockHeight())
	// no-op until real fork migrations are implemented
	ctx.Logger().Info("========== fork logic complete (noop) ==========")
}
