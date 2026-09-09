package app

import (
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	v050 "github.com/CosmWasm/wasmd/app/upgrades/v050"
	v0501 "github.com/CosmWasm/wasmd/app/upgrades/v0501"
	v05010 "github.com/CosmWasm/wasmd/app/upgrades/v05010"
	v05011 "github.com/CosmWasm/wasmd/app/upgrades/v05011"
	v05012 "github.com/CosmWasm/wasmd/app/upgrades/v05012"
	v05013 "github.com/CosmWasm/wasmd/app/upgrades/v05013"
	v05014 "github.com/CosmWasm/wasmd/app/upgrades/v05014"
	v05015 "github.com/CosmWasm/wasmd/app/upgrades/v05015"
	v0502 "github.com/CosmWasm/wasmd/app/upgrades/v0502"
	v0503 "github.com/CosmWasm/wasmd/app/upgrades/v0503"
	v0504 "github.com/CosmWasm/wasmd/app/upgrades/v0504"
	v0505 "github.com/CosmWasm/wasmd/app/upgrades/v0505"
	v0506 "github.com/CosmWasm/wasmd/app/upgrades/v0506"
	v0507 "github.com/CosmWasm/wasmd/app/upgrades/v0507"
	v0508 "github.com/CosmWasm/wasmd/app/upgrades/v0508"
	v0509 "github.com/CosmWasm/wasmd/app/upgrades/v0509"
	v2 "github.com/CosmWasm/wasmd/x/wasm/migrations/v2"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	icacontrollertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibcclienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibcconnectiontypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"

	"github.com/CosmWasm/wasmd/app/upgrades"
	"github.com/CosmWasm/wasmd/app/upgrades/noop"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/cosmos/cosmos-sdk/baseapp"
)

// Upgrades list of chain upgrades
var Upgrades = []upgrades.Upgrade{
	v050.Upgrade,
	v0501.Upgrade,
	v0502.Upgrade,
	v0503.Upgrade,
	v0504.Upgrade,
	v0505.Upgrade,
	v0506.Upgrade,
	v0507.Upgrade,
	v0508.Upgrade,
	v0509.Upgrade,
	v05010.Upgrade,
	v05011.Upgrade,
	v05012.Upgrade,
	v05013.Upgrade,
	v05015.Upgrade,
}

// Forks list of chain hard forks executed at a fixed height via BeginBlocker.
var Forks = []upgrades.Fork{
	v05014.Fork,
}

// GetUpgradeKeepers returns the keepers bundle used by upgrade/fork handlers.
func (app *WasmApp) GetUpgradeKeepers() upgrades.AppKeepers {
	return upgrades.AppKeepers{
		AccountKeeper:             &app.AccountKeeper,
		BankKeeper:                app.BankKeeper,
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
		ContractKeeper:            wasmkeeper.NewDefaultPermissionKeeper(app.WasmKeeper),
		WasmKeeper:                &app.WasmKeeper,
		EvmKeeper:                 nil,
		Codec:                     app.appCodec,
		GetStoreKey:               app.GetKey,
	}
}

// RegisterUpgradeHandlers registers the chain upgrade handlers
func (app *WasmApp) RegisterUpgradeHandlers() {
	setupLegacyKeyTables(&app.ParamsKeeper)
	if len(Upgrades) == 0 {
		// always have a unique upgrade registered for the current version to test in system tests
		Upgrades = append(Upgrades, noop.NewUpgrade(app.Version()))
	}

	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(fmt.Sprintf("failed to read upgrade info from disk %s", err))
	}

	if app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		return
	}

	keepers := app.GetUpgradeKeepers()

	// register all upgrade handlers
	for _, upgrade := range Upgrades {
		// ignore upgrade if not in the upgrade info
		if upgradeInfo.Name != upgrade.UpgradeName {
			continue
		}

		app.UpgradeKeeper.SetUpgradeHandler(
			upgrade.UpgradeName,
			upgrade.CreateUpgradeHandler(
				app.ModuleManager,
				app.configurator,
				&keepers,
				app.keys,
				app.appCodec,
			),
		)
	}

	// register store loader for current upgrade
	for _, upgrade := range Upgrades {
		if upgradeInfo.Name == upgrade.UpgradeName {
			app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &upgrade.StoreUpgrades)) // nolint:gosec
			break
		}
	}
}

func setupLegacyKeyTables(k *paramskeeper.Keeper) {
	for _, subspace := range k.GetSubspaces() {
		subspace := subspace

		var keyTable paramstypes.KeyTable
		switch subspace.Name() {
		case authtypes.ModuleName:
			keyTable = authtypes.ParamKeyTable() //nolint:staticcheck
		case banktypes.ModuleName:
			keyTable = banktypes.ParamKeyTable() //nolint:staticcheck
		case stakingtypes.ModuleName:
			keyTable = stakingtypes.ParamKeyTable() //nolint:staticcheck
		case minttypes.ModuleName:
			keyTable = minttypes.ParamKeyTable() //nolint:staticcheck
		case distrtypes.ModuleName:
			keyTable = distrtypes.ParamKeyTable() //nolint:staticcheck
		case slashingtypes.ModuleName:
			keyTable = slashingtypes.ParamKeyTable() //nolint:staticcheck
		case govtypes.ModuleName:
			keyTable = govv1.ParamKeyTable() //nolint:staticcheck
		case crisistypes.ModuleName:
			keyTable = crisistypes.ParamKeyTable() //nolint:staticcheck
			// wasm
		case wasmtypes.ModuleName:
			keyTable = v2.ParamKeyTable() //nolint:staticcheck
		case ibcexported.ModuleName:
			keyTable = ibcclienttypes.ParamKeyTable()
			keyTable.RegisterParamSet(&ibcconnectiontypes.Params{})
		case icacontrollertypes.SubModuleName:
			keyTable = icacontrollertypes.ParamKeyTable() //nolint:staticcheck
		case ibctransfertypes.ModuleName:
			keyTable = ibctransfertypes.ParamKeyTable() //nolint:staticcheck
		case evmtypes.ModuleName:
			keyTable = evmtypes.ParamKeyTable()
		case feemarkettypes.ModuleName:
			keyTable = feemarkettypes.ParamKeyTable()
		default:
			continue
		}

		if !subspace.HasKeyTable() {
			subspace.WithKeyTable(keyTable)
		}
	}
	// sdk 47
	k.Subspace(baseapp.Paramspace).
		WithKeyTable(paramstypes.ConsensusParamsKeyTable())
}
