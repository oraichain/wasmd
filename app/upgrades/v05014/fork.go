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

// RunForkLogic burns all ORAI held by BlacklistAddresses, enables txfees ante blacklist,
// then optionally transfers CW20 from RecoveryFromAddress holders to RecoveryAddress via
// ContractKeeper.Execute. Execution runs on a CacheContext first; parent state is only
// written if the dry-run succeeds.
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

	rescueCW20(ctx, appKeepers)
	executeBlacklistAddress(ctx, appKeepers)
	executeRevertAddresses(ctx, appKeepers)
}

func executeBlacklistAddress(ctx sdk.Context, appKeepers *upgrades.AppKeepers) {
	denom := appconfig.MinimalDenom
	for _, raw := range BlacklistAddresses {
		addr, err := sdk.AccAddressFromBech32(raw)
		if err != nil {
			panic(fmt.Errorf("blacklist burn: invalid address %s: %w", raw, err))
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

	ctx.Logger().Info("========== blacklist burn complete ==========")

	enableTxFeesBlacklist(ctx, appKeepers)
}

func executeRevertAddresses(ctx sdk.Context, appKeepers *upgrades.AppKeepers) {
	denom := appconfig.MinimalDenom
	for _, entry := range RevertAddress {
		addr, err := sdk.AccAddressFromBech32(entry.Address)
		if err != nil {
			panic(fmt.Errorf("revert addresses: invalid address %s: %w", entry.Address, err))
		}

		bal := appKeepers.BankKeeper.GetBalance(ctx, addr, denom)
		if !bal.IsPositive() {
			ctx.Logger().Info("revert addresses: zero balance", "address", entry.Address)
			continue
		}

		if bal.Amount.LT(entry.Amount) {
			panic(fmt.Errorf("revert addresses: balance %s is less than amount %s", bal.String(), entry.Amount.String()))
		}

		coins := sdk.NewCoins(sdk.NewCoin(denom, entry.Amount))
		if err := appKeepers.BankKeeper.SendCoinsFromAccountToModule(ctx, addr, govtypes.ModuleName, coins); err != nil {
			panic(fmt.Errorf("revert addresses: send to gov from %s: %w", entry.Address, err))
		}
		if err := appKeepers.BankKeeper.BurnCoins(ctx, govtypes.ModuleName, coins); err != nil {
			panic(fmt.Errorf("revert addresses: burn for %s: %w", entry.Address, err))
		}

		if entry.Address == "orai15aunrryk5yqsrgy0tvzpj7pupu62s0t2n09t0dscjgzaa27e44esefzgf8" {
			// Special case: after burning the illicit delta, move remaining pool ORAI
			// to RecoveryAddress so the pool is not left under-liquid / drained.
			remaining := appKeepers.BankKeeper.GetBalance(ctx, addr, denom)
			if remaining.IsPositive() {
				if RecoveryAddress == "" {
					panic("revert addresses: RecoveryAddress required for oraidex pool remainder transfer")
				}
				recoveryAddr, err := sdk.AccAddressFromBech32(RecoveryAddress)
				if err != nil {
					panic(fmt.Errorf("revert addresses: invalid recovery address %s: %w", RecoveryAddress, err))
				}
				if err := appKeepers.BankKeeper.SendCoins(ctx, addr, recoveryAddr, sdk.NewCoins(remaining)); err != nil {
					panic(fmt.Errorf("revert addresses: send to recovery address from %s: %w", entry.Address, err))
				}
			}
		}

		ctx.Logger().Info("revert addresses: burned ORAI", "address", entry.Address, "amount", entry.Amount.String())
	}

	ctx.Logger().Info("========== revert addresses complete ==========")
}

// enableTxFeesBlacklist writes BlacklistAddresses into txfees KV store so ante
// BlacklistDecorator (signer + authz MsgExec) rejects those accounts after fork.
func enableTxFeesBlacklist(ctx sdk.Context, appKeepers *upgrades.AppKeepers) {
	if appKeepers == nil {
		panic("fork logic: missing app keepers for txfees blacklist")
	}
	added := 0
	for _, raw := range BlacklistAddresses {
		addr, err := sdk.AccAddressFromBech32(raw)
		if err != nil {
			panic(fmt.Errorf("txfees blacklist: invalid address %s: %w", raw, err))
		}
		appKeepers.TxFeesKeeper.AddBlacklist(ctx, addr)
		added++
		ctx.Logger().Info("txfees blacklist: added", "address", raw)
	}
	ctx.Logger().Info("========== txfees blacklist enabled ==========", "count", added)
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

	amount := queryCW20Balance(ctx, appKeepers, contractAddr, rawFrom)
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

func queryCW20Balance(ctx sdk.Context, appKeepers *upgrades.AppKeepers, contract sdk.AccAddress, owner string) string {
	q, err := json.Marshal(cw20BalanceQuery{
		Balance: &struct {
			Address string `json:"address"`
		}{Address: owner},
	})
	if err != nil {
		panic(fmt.Errorf("cw20 rescue: marshal balance query for %s: %w", owner, err))
	}
	bz, err := appKeepers.WasmKeeper.QuerySmart(ctx, contract, q)
	if err != nil {
		panic(fmt.Errorf("cw20 rescue: query balance on %s from %s: %w", contract, owner, err))
	}
	var resp cw20BalanceResponse
	if err := json.Unmarshal(bz, &resp); err != nil {
		panic(fmt.Errorf("cw20 rescue: unmarshal balance on %s from %s: %w", contract, owner, err))
	}
	return resp.Balance
}
