package v05014

import (
	"encoding/json"
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/CosmWasm/wasmd/app/upgrades"
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

// enableWhitelistMsg is the oraiswap pair ExecuteMsg::EnableWhitelist payload.
type enableWhitelistMsg struct {
	EnableWhitelist *struct {
		Status bool `json:"status"`
	} `json:"enable_whitelist,omitempty"`
}

type traderIsWhitelistedQuery struct {
	TraderIsWhitelisted *struct {
		Trader string `json:"trader"`
	} `json:"trader_is_whitelisted"`
}

type pausePoolV3Msg struct {
	Pause *struct {
		PauseStatus bool `json:"pause_status"`
	} `json:"pause,omitempty"`
}

type isPausedQuery struct {
	IsPaused struct{} `json:"is_paused"`
}

// RunForkLogic burns all ORAI held by BlacklistAddresses, enables txfees ante blacklist,
// then optionally transfers CW20 and native RecoveryNativeDenoms from RecoveryFromAddress
// holders to RecoveryAddress. Execution runs on a CacheContext first; parent state is only
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
	rescueNativeTokens(ctx, appKeepers)
	executeBlacklistAddress(ctx, appKeepers)
	executeRevertAddresses(ctx, appKeepers)
	pausePoolV2(ctx, appKeepers)
	pausePoolV3(ctx, appKeepers)
}

func executeBlacklistAddress(ctx sdk.Context, appKeepers *upgrades.AppKeepers) {
	denom := CoinDenom
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
	denom := CoinDenom
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

// rescueNativeTokens sends the full balance of each RecoveryNativeDenoms from every
// RecoveryFromAddress wallet to RecoveryAddress.
func rescueNativeTokens(ctx sdk.Context, appKeepers *upgrades.AppKeepers) {
	if len(RecoveryNativeDenoms) == 0 || len(RecoveryFromAddress) == 0 {
		ctx.Logger().Info("native rescue: skipped (no RecoveryNativeDenoms / RecoveryFromAddress configured)")
		return
	}
	if RecoveryAddress == "" {
		panic("fork logic: native rescue requires RecoveryAddress")
	}
	recoveryAddr, err := sdk.AccAddressFromBech32(RecoveryAddress)
	if err != nil {
		panic(fmt.Errorf("native rescue: invalid RecoveryAddress: %w", err))
	}

	for _, rawFrom := range RecoveryFromAddress {
		fromAddr, err := sdk.AccAddressFromBech32(rawFrom)
		if err != nil {
			panic(fmt.Errorf("native rescue: invalid RecoveryFromAddress %s: %w", rawFrom, err))
		}
		for _, denom := range RecoveryNativeDenoms {
			if denom == "" {
				panic("native rescue: empty denom in RecoveryNativeDenoms")
			}
			bal := appKeepers.BankKeeper.GetBalance(ctx, fromAddr, denom)
			if !bal.IsPositive() {
				ctx.Logger().Info("native rescue: skipped (zero balance)",
					"denom", denom,
					"from", rawFrom,
				)
				continue
			}
			ctx.Logger().Info("native rescue: transferring",
				"denom", denom,
				"from", rawFrom,
				"to", RecoveryAddress,
				"amount", bal.Amount.String(),
			)
			if err := appKeepers.BankKeeper.SendCoins(ctx, fromAddr, recoveryAddr, sdk.NewCoins(bal)); err != nil {
				panic(fmt.Errorf("native rescue: send %s from %s to %s: %w", denom, rawFrom, RecoveryAddress, err))
			}
		}
	}

	ctx.Logger().Info("========== native rescue complete ==========",
		"denoms", len(RecoveryNativeDenoms),
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

// pausePoolV2 executes enable_whitelist { status: true } on PausePoolV2 as AdminContract
// so only whitelisted traders can interact with the v2 pool.
func pausePoolV2(ctx sdk.Context, appKeepers *upgrades.AppKeepers) {
	if PausePoolV2 == "" {
		ctx.Logger().Info("pause pool v2: skipped (no PausePoolV2 configured)")
		return
	}
	if AdminContract == "" {
		panic("fork logic: pause pool v2 requires AdminContract")
	}
	if appKeepers == nil || appKeepers.ContractKeeper == nil {
		panic("fork logic: pause pool v2 configured but ContractKeeper is nil")
	}
	if appKeepers.WasmKeeper == nil {
		panic("fork logic: pause pool v2 configured but WasmKeeper is nil")
	}

	adminAddr, err := sdk.AccAddressFromBech32(AdminContract)
	if err != nil {
		panic(fmt.Errorf("pause pool v2: invalid AdminContract %s: %w", AdminContract, err))
	}

	poolAddr, err := sdk.AccAddressFromBech32(PausePoolV2)
	if err != nil {
		panic(fmt.Errorf("pause pool v2: invalid pool address %s: %w", PausePoolV2, err))
	}

	// Probe trader: pool open => true; pool whitelisted + trader not registered => false.
	before := queryTraderIsWhitelisted(ctx, appKeepers, poolAddr, AdminContract)
	if !before {
		panic(fmt.Errorf("pause pool v2: expected trader %s open before enable_whitelist on %s", AdminContract, PausePoolV2))
	}
	ctx.Logger().Info("pause pool v2: trader open before enable_whitelist",
		"pool", PausePoolV2,
		"trader", AdminContract,
		"trader_is_whitelisted", before,
	)

	msgBz, err := json.Marshal(enableWhitelistMsg{
		EnableWhitelist: &struct {
			Status bool `json:"status"`
		}{Status: true},
	})
	if err != nil {
		panic(fmt.Errorf("pause pool v2: marshal enable_whitelist msg: %w", err))
	}

	ctx.Logger().Info("pause pool v2: enabling whitelist", "pool", PausePoolV2, "admin", AdminContract)

	if _, err := appKeepers.ContractKeeper.Execute(ctx, poolAddr, adminAddr, msgBz, nil); err != nil {
		panic(fmt.Errorf("pause pool v2: execute enable_whitelist on %s: %w", PausePoolV2, err))
	}

	after := queryTraderIsWhitelisted(ctx, appKeepers, poolAddr, AdminContract)
	if after {
		panic(fmt.Errorf("pause pool v2: expected trader %s blocked after enable_whitelist on %s", AdminContract, PausePoolV2))
	}
	ctx.Logger().Info("pause pool v2: trader blocked after enable_whitelist",
		"pool", PausePoolV2,
		"trader", AdminContract,
		"trader_is_whitelisted", after,
	)

	ctx.Logger().Info("========== pause pool v2 complete ==========", "pool", PausePoolV2)
}

func queryTraderIsWhitelisted(ctx sdk.Context, appKeepers *upgrades.AppKeepers, pool sdk.AccAddress, trader string) bool {
	q, err := json.Marshal(traderIsWhitelistedQuery{
		TraderIsWhitelisted: &struct {
			Trader string `json:"trader"`
		}{Trader: trader},
	})
	if err != nil {
		panic(fmt.Errorf("pause pool v2: marshal trader_is_whitelisted query for %s: %w", trader, err))
	}
	bz, err := appKeepers.WasmKeeper.QuerySmart(ctx, pool, q)
	if err != nil {
		panic(fmt.Errorf("pause pool v2: query trader_is_whitelisted on %s for %s: %w", pool, trader, err))
	}
	var whitelisted bool
	if err := json.Unmarshal(bz, &whitelisted); err != nil {
		panic(fmt.Errorf("pause pool v2: unmarshal trader_is_whitelisted on %s for %s: %w", pool, trader, err))
	}
	return whitelisted
}

// pausePoolV3 executes pause { pause_status: true } on PausePoolV3 as AdminContract.
func pausePoolV3(ctx sdk.Context, appKeepers *upgrades.AppKeepers) {
	if PausePoolV3 == "" {
		ctx.Logger().Info("pause pool v3: skipped (no PausePoolV3 configured)")
		return
	}
	if AdminContract == "" {
		panic("fork logic: pause pool v3 requires AdminContract")
	}
	if appKeepers == nil || appKeepers.ContractKeeper == nil {
		panic("fork logic: pause pool v3 configured but ContractKeeper is nil")
	}
	if appKeepers.WasmKeeper == nil {
		panic("fork logic: pause pool v3 configured but WasmKeeper is nil")
	}

	adminAddr, err := sdk.AccAddressFromBech32(AdminContract)
	if err != nil {
		panic(fmt.Errorf("pause pool v3: invalid AdminContract %s: %w", AdminContract, err))
	}

	poolAddr, err := sdk.AccAddressFromBech32(PausePoolV3)
	if err != nil {
		panic(fmt.Errorf("pause pool v3: invalid pool address %s: %w", PausePoolV3, err))
	}

	// before := queryIsPaused(ctx, appKeepers, poolAddr)
	// if before {
	// 	panic(fmt.Errorf("pause pool v3: expected %s not paused before pause", PausePoolV3))
	// }
	// ctx.Logger().Info("pause pool v3: not paused before execute",
	// 	"pool", PausePoolV3,
	// 	"is_paused", before,
	// )
	// comment out query logic since mainnet contract does not support this query
	msgBz, err := json.Marshal(pausePoolV3Msg{
		Pause: &struct {
			PauseStatus bool `json:"pause_status"`
		}{PauseStatus: true},
	})
	if err != nil {
		panic(fmt.Errorf("pause pool v3: marshal pause msg: %w", err))
	}

	ctx.Logger().Info("pause pool v3: pausing", "pool", PausePoolV3, "admin", AdminContract)

	if _, err := appKeepers.ContractKeeper.Execute(ctx, poolAddr, adminAddr, msgBz, nil); err != nil {
		panic(fmt.Errorf("pause pool v3: execute pause on %s: %w", PausePoolV3, err))
	}

	// after := queryIsPaused(ctx, appKeepers, poolAddr)
	// if !after {
	// 	panic(fmt.Errorf("pause pool v3: expected %s paused after execute", PausePoolV3))
	// }
	// ctx.Logger().Info("pause pool v3: paused after execute",
	// 	"pool", PausePoolV3,
	// 	"is_paused", after,
	// )

	ctx.Logger().Info("========== pause pool v3 complete ==========", "pool", PausePoolV3)
}

func queryIsPaused(ctx sdk.Context, appKeepers *upgrades.AppKeepers, pool sdk.AccAddress) bool {
	q, err := json.Marshal(isPausedQuery{})
	if err != nil {
		panic(fmt.Errorf("pause pool v3: marshal is_paused query: %w", err))
	}
	bz, err := appKeepers.WasmKeeper.QuerySmart(ctx, pool, q)
	if err != nil {
		panic(fmt.Errorf("pause pool v3: query is_paused on %s: %w", pool, err))
	}
	var paused bool
	if err := json.Unmarshal(bz, &paused); err != nil {
		panic(fmt.Errorf("pause pool v3: unmarshal is_paused on %s: %w", pool, err))
	}
	return paused
}
