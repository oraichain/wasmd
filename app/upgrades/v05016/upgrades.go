package v05016

import (
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/CosmWasm/wasmd/app/upgrades"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

// UpgradeName defines the on-chain upgrade name
const UpgradeName = "v0.50.16"

// CompromisedBlacklistAddresses are accounts under active attack (attacker
// undelegating to drain funds). Added to the txfees blacklist during this
// upgrade so that:
//
//   - Outbound bank sends from these addresses are blocked (app bank send
//     restriction + ante BlacklistDecorator).
//   - Inbound credits remain allowed — staking CompletingUnbonding (EndBlocker)
//     still sends unbonded tokens TO these addresses after the unbonding
//     period. Funds land as expected, then cannot be moved out.
//
// New user-signed txs from these accounts (bank send, withdraw rewards, new
// undelegate, etc.) are rejected by ante after the upgrade.
var CompromisedBlacklistAddresses = []string{
	"orai1rhza73qeztj0am3qszgxm7p7tfm5m5gdevdaa9",
	"orai1td0lh3398kerk4tfl7gs30acs50tanhl4qqap8",
}

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
	return func(goCtx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(goCtx)
		ctx.Logger().Info("starting upgrade", "upgrade_name", UpgradeName)

		migrationRes, err := mm.RunMigrations(ctx, configurator, fromVM)
		if err != nil {
			return nil, err
		}

		if err := addCompromisedBlacklist(ctx, ak); err != nil {
			return nil, err
		}

		ctx.Logger().Info("completed upgrade", "upgrade_name", UpgradeName)
		return migrationRes, nil
	}
}

func addCompromisedBlacklist(ctx sdk.Context, ak *upgrades.AppKeepers) error {
	if ak == nil {
		return fmt.Errorf("%s: missing app keepers for txfees blacklist", UpgradeName)
	}

	added := 0
	for _, raw := range CompromisedBlacklistAddresses {
		addr, err := sdk.AccAddressFromBech32(raw)
		if err != nil {
			return fmt.Errorf("%s: invalid blacklist address %s: %w", UpgradeName, raw, err)
		}

		already, err := ak.TxFeesKeeper.IsBlacklisted(ctx, addr)
		if err != nil {
			return fmt.Errorf("%s: check blacklist %s: %w", UpgradeName, raw, err)
		}
		if already {
			ctx.Logger().Info("txfees blacklist: already present, skip", "address", raw)
			continue
		}

		if err := ak.TxFeesKeeper.AddBlacklist(ctx, addr); err != nil {
			return fmt.Errorf("%s: add blacklist %s: %w", UpgradeName, raw, err)
		}
		added++
		ctx.Logger().Info("txfees blacklist: added compromised address", "address", raw)
	}

	ctx.Logger().Info("========== compromised blacklist enabled ==========",
		"upgrade", UpgradeName,
		"added", added,
		"total_configured", len(CompromisedBlacklistAddresses),
	)
	return nil
}
