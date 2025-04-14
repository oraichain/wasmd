package v050

import (
	"context"
	"errors"

	storetypes "cosmossdk.io/store/types"
	circuittypes "cosmossdk.io/x/circuit/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmtbfttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
	erc20types "github.com/evmos/ethermint/x/erc20/types"

	"github.com/CosmWasm/wasmd/app/upgrades"
	"github.com/cosmos/cosmos-sdk/x/group"
)

// UpgradeName defines the on-chain upgrade name
const UpgradeName = "v0.50.0"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: storetypes.StoreUpgrades{
		Added: []string{
			circuittypes.ModuleName,
			consensustypes.StoreKey,
			crisistypes.StoreKey,
			erc20types.StoreKey,
			group.StoreKey,
		},
		Deleted: []string{"utilevm", "evmutil", "intertx"},
	},
}

func CreateUpgradeHandler(
	mm upgrades.ModuleManager,
	configurator module.Configurator,
	ak *upgrades.AppKeepers,
	keys map[string]*storetypes.KVStoreKey,
	cdc codec.BinaryCodec,
) upgradetypes.UpgradeHandler {
	// sdk 47 to sdk 50
	return func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {

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
		err := ak.ConsensusParamsKeeper.ParamsStore.Set(ctx, cp)
		if err != nil {
			return nil, err
		}

		// actually update consensus param keeper store
		Authority := authtypes.NewModuleAddress(govtypes.ModuleName)
		AuthorityAddr := Authority.String()
		updateConsensusParamStore := consensustypes.MsgUpdateParams{Authority: AuthorityAddr, Block: cp.Block, Evidence: cp.Evidence, Validator: cp.Validator, Abci: cp.Abci}
		_, err = ak.ConsensusParamsKeeper.UpdateParams(ctx, &updateConsensusParamStore)
		if err != nil {
			return nil, err
		}

		sdkCtx := sdk.UnwrapSDKContext(ctx)

		// upgrade ica capability
		err = UpgradeIcaController(sdkCtx, ak.IBCKeeper, ak.ScopedICAControllerKeeper)
		if err != nil {
			panic(err)
		}

		// upgrade mint module params
		err = UpgradeMintParams(sdkCtx, ak.ParamsKeeper)
		if err != nil {
			panic(err)
		}

		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}

func UpgradeIcaController(ctx sdk.Context, ibcKeeper *ibckeeper.Keeper, scopedICAControllerKeeper *capabilitykeeper.ScopedKeeper) error {
	chanels := ibcKeeper.ChannelKeeper.GetAllChannelsWithPortPrefix(ctx, icatypes.ControllerPortPrefix)
	for _, ch := range chanels {
		name := host.ChannelCapabilityPath(ch.PortId, ch.ChannelId)
		_, found := scopedICAControllerKeeper.GetCapability(ctx, name)

		// if not found then try to add capability for chanel
		if !found {
			_, err := scopedICAControllerKeeper.NewCapability(ctx, name)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func UpgradeMintParams(ctx sdk.Context, paramsKeeper *paramskeeper.Keeper) error {
	mintSpace, exist := paramsKeeper.GetSubspace(minttypes.ModuleName)
	if !exist {
		return errors.New("mint space must existed")
	}

	var mintParams minttypes.Params
	mintSpace.GetParamSet(ctx, &mintParams)
	if mintParams.InflationMin.GT(mintParams.InflationMax) {
		mintParams.InflationMin = mintParams.InflationMax
		mintSpace.SetParamSet(ctx, &mintParams)
	}

	return nil
}
