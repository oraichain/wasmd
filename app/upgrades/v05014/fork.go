package v10

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/CosmWasm/wasmd/app/upgrades"
	appconfig "github.com/CosmWasm/wasmd/cmd/config"
)

// BlacklistAddresses are wallets whose ORAI is burned at ForkHeight and blocked from bank sends
// (see app.RegisterBankSendRestrictions / app.BlacklistSendRestriction).
var BlacklistAddresses = []string{
	"orai1hru4a5w0c29wr36l2dgaymqqd4h0vju9tlvk8w",
	"orai1ycryq0mghafwfwr346d5qe2ce8zvmflns08khy",
	"orai1ahug8tgnw7k78dj5567n0cch9l86ntsdtwv8z3sdd3g4jzqpg7qqqmr9fl",
	"orai1cm9xkysr58kgcvzjs0shlgma557hsu62s76f98",
	"orai15aunrryk5yqsrgy0tvzpj7pupu62s0t2n09t0dscjgzaa27e44esefzgf8",
}

// RunForkLogic burns all ORAI held by BlacklistAddresses.
// Send blacklist enforcement starts at height > ForkHeight (app.BlacklistSendRestriction).
func RunForkLogic(ctx sdk.Context, appKeepers *upgrades.AppKeepers) {
	ctx.Logger().Info("========== running v0.50.14 fork logic ==========", "height", ctx.BlockHeight())

	if appKeepers == nil || appKeepers.BankKeeper == nil {
		panic("fork logic: missing bank keeper")
	}

	denom := appconfig.MinimalDenom
	for _, raw := range BlacklistAddresses {
		addr, err := sdk.AccAddressFromBech32(raw)
		if err != nil {
			ctx.Logger().Error("blacklist burn: skip invalid address", "address", raw, "err", err)
			continue
		}

		bal := appKeepers.BankKeeper.GetBalance(ctx, addr, denom)
		if !bal.IsPositive() {
			ctx.Logger().Info("blacklist burn: zero balance", "address", raw)
			continue
		}

		coins := sdk.NewCoins(bal)
		if err := appKeepers.BankKeeper.SendCoinsFromAccountToModule(ctx, addr, govtypes.ModuleName, coins); err != nil {
			panic(fmt.Errorf("blacklist burn: send to gov from %s: %w", raw, err))
		}
		if err := appKeepers.BankKeeper.BurnCoins(ctx, govtypes.ModuleName, coins); err != nil {
			panic(fmt.Errorf("blacklist burn: burn for %s: %w", raw, err))
		}

		ctx.Logger().Info("blacklist burn: burned ORAI", "address", raw, "amount", bal.String())
	}

	ctx.Logger().Info("========== fork burn complete ==========")
}
