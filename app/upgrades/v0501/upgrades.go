package v050

import (
	"context"
	"errors"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
	v6 "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/migrations/v6"

	"github.com/CosmWasm/wasmd/app/upgrades"
	"github.com/CosmWasm/wasmd/cmd/config"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	gov1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	channelkeeper "github.com/cosmos/ibc-go/v8/modules/core/04-channel/keeper"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
)

// UpgradeName defines the on-chain upgrade name
const UpgradeName = "v0.50.1"

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
	return func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {

		sdkCtx := sdk.UnwrapSDKContext(ctx)

		if err := v6.MigrateICS27ChannelCapability(sdkCtx, cdc, keys[capabilitytypes.ModuleName], ak.CapabilityKeeper, "intertx"); err != nil {
			return nil, err
		}

		if err := ReleaseWrongIcaControllerCaps(sdkCtx, ak.IBCKeeper.ChannelKeeper, ak.ScopedICAControllerKeeper, ak.ScopedIBCKeeper); err != nil {
			return nil, err
		}

		if err := UpgradeMintParams(sdkCtx, ak.ParamsKeeper, ak.MintKeeper); err != nil {
			return nil, err
		}

		if err := upgradeGovParams(sdkCtx, ak.GovKeeper); err != nil {
			return nil, err
		}

		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}

func ReleaseWrongIcaControllerCaps(ctx sdk.Context, channelKeeper channelkeeper.Keeper, scopedICAControllerKeeper *capabilitykeeper.ScopedKeeper, scopedIBCKeeper *capabilitykeeper.ScopedKeeper) error {
	chanels := channelKeeper.GetAllChannelsWithPortPrefix(ctx, icatypes.ControllerPortPrefix)
	for _, ch := range chanels {
		name := host.ChannelCapabilityPath(ch.PortId, ch.ChannelId)
		cap, found := scopedICAControllerKeeper.GetCapability(ctx, name)
		if found {
			if err := scopedICAControllerKeeper.ReleaseCapability(ctx, cap); err != nil {
				return err
			}
		}

		ibcCap, ibcCapFound := scopedIBCKeeper.GetCapability(ctx, name)
		if found && ibcCapFound {
			if err := scopedICAControllerKeeper.SetCapability(ctx, ibcCap, name); err != nil {
				return err
			}
		}
	}

	return nil
}

func upgradeGovParams(ctx sdk.Context, govKeeper *govkeeper.Keeper) error {

	govParams := gov1.DefaultParams()
	govParams.BurnVoteVeto = true
	govParams.ExpeditedMinDeposit = sdk.NewCoins(sdk.NewCoin(config.MinimalDenom, gov1.DefaultMinExpeditedDepositTokens))
	govParams.MinDeposit = sdk.NewCoins(sdk.NewCoin(config.MinimalDenom, gov1.DefaultMinDepositTokens))
	votingPeriod := time.Hour * 24 * 5 // 5 days
	depositPeriod := time.Hour * 24 * 7 // 7 days
	govParams.VotingPeriod = &votingPeriod
	govParams.MaxDepositPeriod = &depositPeriod

	govKeeper.Params.Set(ctx, govParams)

	return nil
}

func UpgradeMintParams(ctx sdk.Context, paramsKeeper *paramskeeper.Keeper, mintKeeper *mintkeeper.Keeper) error {

	mintParams := minttypes.DefaultParams()
	mintParams.BlocksPerYear = 39420000
	mintParams.GoalBonded = math.LegacyMustNewDecFromStr("0.67")
	mintParams.InflationRateChange = math.LegacyMustNewDecFromStr("0.13")
	mintParams.InflationMin = math.LegacyMustNewDecFromStr("0.085")
	mintParams.InflationMax = mintParams.InflationMin
	mintParams.MintDenom = config.MinimalDenom

	mintSpace, exist := paramsKeeper.GetSubspace(minttypes.ModuleName)
	if !exist {
		return errors.New("mint space must existed")
	}
	mintSpace.SetParamSet(ctx, &mintParams)

	mintKeeper.Params.Set(ctx, mintParams)

	return nil
}
