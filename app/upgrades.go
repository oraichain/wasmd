package app

import (
	"context"
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	cmtbfttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	icacontrollertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibcclienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibcconnectiontypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	feemarkettypes "github.com/evmos/ethermint/x/feemarket/types"

	"github.com/CosmWasm/wasmd/app/upgrades"
	"github.com/CosmWasm/wasmd/app/upgrades/noop"
	v050 "github.com/CosmWasm/wasmd/app/upgrades/v050"
	v2 "github.com/CosmWasm/wasmd/x/wasm/migrations/v2"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
)

// Upgrades list of chain upgrades
var Upgrades = []upgrades.Upgrade{v050.Upgrade}

// RegisterUpgradeHandlers registers the chain upgrade handlers
func (app *WasmApp) RegisterUpgradeHandlers() {
	setupLegacyKeyTables(&app.ParamsKeeper)
	if len(Upgrades) == 0 {
		// always have a unique upgrade registered for the current version to test in system tests
		Upgrades = append(Upgrades, noop.NewUpgrade(app.Version()))
	}

	keepers := upgrades.AppKeepers{
		AccountKeeper:         &app.AccountKeeper,
		ParamsKeeper:          &app.ParamsKeeper,
		ConsensusParamsKeeper: &app.ConsensusParamsKeeper,
		CapabilityKeeper:      app.CapabilityKeeper,
		IBCKeeper:             app.IBCKeeper,
		Codec:                 app.appCodec,
		GetStoreKey:           app.GetKey,
	}

	app.GetStoreKeys()
	// register all upgrade handlers
	for _, upgrade := range Upgrades {
		if upgrade.UpgradeName == v050.Upgrade.UpgradeName {
			app.UpgradeKeeper.SetUpgradeHandler(
				upgrade.UpgradeName,
				func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {

					// special case, we need to resolve this issue: https://github.com/cosmos/cosmos-sdk/issues/20160
					defaultConsensusParams := cmtbfttypes.DefaultConsensusParams()
					cp := cmtproto.ConsensusParams{
						Block: &cmtproto.BlockParams{
							// hard-coded max bytes like in prod params
							MaxBytes: 1048576,
							MaxGas:   defaultConsensusParams.Block.MaxGas,
						},
						Evidence: &cmtproto.EvidenceParams{
							MaxAgeNumBlocks: defaultConsensusParams.Evidence.MaxAgeNumBlocks,
							MaxAgeDuration:  defaultConsensusParams.Evidence.MaxAgeDuration,
							MaxBytes:        defaultConsensusParams.Evidence.MaxBytes,
						},
						Validator: &cmtproto.ValidatorParams{
							PubKeyTypes: defaultConsensusParams.Validator.PubKeyTypes,
						},
						Version: defaultConsensusParams.ToProto().Version, // Version is stored in x/upgrade
					}
					err := app.ConsensusParamsKeeper.ParamsStore.Set(ctx, cp)
					if err != nil {
						return nil, err
					}

					// actually update consensus param keeper store
					Authority := authtypes.NewModuleAddress(govtypes.ModuleName)
					AuthorityAddr := Authority.String()
					updateConsensusParamStore := consensustypes.MsgUpdateParams{Authority: AuthorityAddr, Block: cp.Block, Evidence: cp.Evidence, Validator: cp.Validator, Abci: cp.Abci}
					_, err = app.ConsensusParamsKeeper.UpdateParams(ctx, &updateConsensusParamStore)
					if err != nil {
						return nil, err
					}

					sdkCtx := sdk.UnwrapSDKContext(ctx)

					// upgrade ica capability
					chanels := app.IBCKeeper.ChannelKeeper.GetAllChannelsWithPortPrefix(sdkCtx, icatypes.ControllerPortPrefix)
					for _, ch := range chanels {
						name := host.ChannelCapabilityPath(ch.PortId, ch.ChannelId)
						_, found := app.ScopedICAControllerKeeper.GetCapability(sdkCtx, name)

						// if not found then try to add capability for chanel
						if !found {
							_, err := app.ScopedICAControllerKeeper.NewCapability(sdkCtx, name)
							if err != nil {
								panic(err)
							}
						}
					}

					// upgrade mint module params
					mintSpace, exist := app.ParamsKeeper.GetSubspace(minttypes.ModuleName)
					if !exist {
						panic("mint module must have subspace")
					}

					var mintParams minttypes.Params
					mintSpace.GetParamSet(sdkCtx, &mintParams)
					mintParams.InflationMin = mintParams.InflationMax
					mintSpace.SetParamSet(sdkCtx, &mintParams)

					return app.ModuleManager.RunMigrations(ctx, app.configurator, fromVM)
				},
			)
			continue
		}

		app.UpgradeKeeper.SetUpgradeHandler(
			upgrade.UpgradeName,
			upgrade.CreateUpgradeHandler(
				app.ModuleManager,
				app.configurator,
				&keepers,
			),
		)
	}

	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(fmt.Sprintf("failed to read upgrade info from disk %s", err))
	}

	if app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		return
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
