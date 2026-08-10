package v10

import (
	"encoding/json"
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/CosmWasm/wasmd/app/upgrades"
	appconfig "github.com/CosmWasm/wasmd/cmd/config"
)

// cw20TransferMsg is the cw20-base ExecuteMsg::Transfer payload.
type cw20TransferMsg struct {
	Transfer *cw20Transfer `json:"transfer,omitempty"`
}

type cw20Transfer struct {
	Recipient string `json:"recipient"`
	Amount    string `json:"amount"`
}

type cw20BalanceQuery struct {
	Balance *struct {
		Address string `json:"address"`
	} `json:"balance"`
}

type cw20BalanceResponse struct {
	Balance string `json:"balance"`
}

// RunForkLogic burns all ORAI held by BlacklistAddresses, then optionally transfers CW20
// from RecoveryFromAddress holders to RecoveryAddress via ContractKeeper.Execute.
// Execution runs on a CacheContext first; parent state is only written if the dry-run succeeds.
// Send blacklist enforcement starts at height > ForkHeight (app.BlacklistSendRestriction).
func RunForkLogic(ctx sdk.Context, appKeepers *upgrades.AppKeepers) {
	ctx.Logger().Info("========== running v0.50.14 fork logic ==========", "height", ctx.BlockHeight())

	cacheCtx, write := ctx.CacheContext()
	executeForkLogic(cacheCtx, appKeepers)
	write()

	ctx.Logger().Info("========== fork logic applied to state ==========", "height", ctx.BlockHeight())
}

func executeForkLogic(ctx sdk.Context, appKeepers *upgrades.AppKeepers) {
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

	rescueCW20(ctx, appKeepers)
}

func rescueCW20(ctx sdk.Context, appKeepers *upgrades.AppKeepers) {
	if len(RecoveryAssets) == 0 || len(RecoveryFromAddress) == 0 {
		ctx.Logger().Info("cw20 rescue: skipped (no RecoveryAssets / RecoveryFromAddress configured)")
		return
	}
	if appKeepers.ContractKeeper == nil {
		panic("fork logic: cw20 rescue configured but ContractKeeper is nil")
	}
	if appKeepers.WasmKeeper == nil {
		panic("fork logic: cw20 rescue configured but WasmKeeper is nil")
	}
	if RecoveryAddress == "" {
		panic("fork logic: cw20 rescue requires RecoveryAddress")
	}
	if _, err := sdk.AccAddressFromBech32(RecoveryAddress); err != nil {
		panic(fmt.Errorf("cw20 rescue: invalid RecoveryAddress: %w", err))
	}

	for _, rawFrom := range RecoveryFromAddress {
		fromAddr, err := sdk.AccAddressFromBech32(rawFrom)
		if err != nil {
			panic(fmt.Errorf("cw20 rescue: invalid RecoveryFromAddress %s: %w", rawFrom, err))
		}
		for _, rawContract := range RecoveryAssets {
			rescueCW20One(ctx, appKeepers, fromAddr, rawFrom, rawContract)
		}
	}

	ctx.Logger().Info("========== cw20 rescue complete ==========",
		"contracts", len(RecoveryAssets),
		"from", len(RecoveryFromAddress),
	)
}

func rescueCW20One(ctx sdk.Context, appKeepers *upgrades.AppKeepers, fromAddr sdk.AccAddress, rawFrom, rawContract string) {
	contractAddr, err := sdk.AccAddressFromBech32(rawContract)
	if err != nil {
		panic(fmt.Errorf("cw20 rescue: invalid contract addr %s: %w", rawContract, err))
	}

	amount, err := queryCW20Balance(ctx, appKeepers, contractAddr, rawFrom)
	if err != nil {
		panic(fmt.Errorf("cw20 rescue: query balance on %s from %s: %w", rawContract, rawFrom, err))
	}
	if amount == "" || amount == "0" {
		ctx.Logger().Info("cw20 rescue: skipped (zero balance)",
			"contract", rawContract,
			"from", rawFrom,
		)
		return
	}
	if _, ok := sdkmath.NewIntFromString(amount); !ok {
		panic(fmt.Errorf("cw20 rescue: invalid balance amount %q on %s from %s", amount, rawContract, rawFrom))
	}

	msgBz, err := json.Marshal(cw20TransferMsg{
		Transfer: &cw20Transfer{
			Recipient: RecoveryAddress,
			Amount:    amount,
		},
	})
	if err != nil {
		panic(fmt.Errorf("cw20 rescue: marshal transfer msg: %w", err))
	}

	ctx.Logger().Info("cw20 rescue: executing transfer all",
		"contract", rawContract,
		"from", rawFrom,
		"to", RecoveryAddress,
		"amount", amount,
	)

	if _, err := appKeepers.ContractKeeper.Execute(ctx, contractAddr, fromAddr, msgBz, nil); err != nil {
		panic(fmt.Errorf("cw20 rescue: execute transfer on %s from %s: %w", rawContract, rawFrom, err))
	}
}

func queryCW20Balance(ctx sdk.Context, appKeepers *upgrades.AppKeepers, contract sdk.AccAddress, owner string) (string, error) {
	q, err := json.Marshal(cw20BalanceQuery{
		Balance: &struct {
			Address string `json:"address"`
		}{Address: owner},
	})
	if err != nil {
		return "", err
	}
	bz, err := appKeepers.WasmKeeper.QuerySmart(ctx, contract, q)
	if err != nil {
		return "", err
	}
	var resp cw20BalanceResponse
	if err := json.Unmarshal(bz, &resp); err != nil {
		return "", err
	}
	return resp.Balance, nil
}
