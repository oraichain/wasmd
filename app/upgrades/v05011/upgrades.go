package v05011

import (
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/CosmWasm/wasmd/app/upgrades"
	evmostypes "github.com/CosmWasm/wasmd/app/upgrades/v05011/types"
	cmn "github.com/CosmWasm/wasmd/precompile/common"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	ethsecp256k1 "github.com/cosmos/evm/crypto/ethsecp256k1"
	evmkeeper "github.com/cosmos/evm/x/vm/keeper"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
)

// UpgradeName defines the on-chain upgrade name
const UpgradeName = "v0.50.11"

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
		logger := ctx.Logger().With("upgrade", UpgradeName)

		// run migrations first so it will not override our upgrade logic
		migrationRes, err := mm.RunMigrations(ctx, configurator, fromVM)
		if err != nil {
			return migrationRes, err
		}

		// We need to migrate the EthAccounts to BaseAccounts
		logger.Info("==========migrating EthAccounts to BaseAccounts==========")
		MigrateEthAccountsToBaseAccounts(ctx, *ak.AccountKeeper, ak.EvmKeeper)
		logger.Info("=========================================================")
		// Set actives precompiles
		logger.Info("=============Set actives precompile contracts============")
		ActivateStaticPrecompiles(ctx, ak.EvmKeeper)
		logger.Info("=========================================================")

		return migrationRes, err
	}
}

// MigrateEthAccountsToBaseAccounts is used to store the code hash of the associated
// smart contracts in the dedicated store in the EVM module and convert the former
// EthAccounts to standard Cosmos SDK accounts.
func MigrateEthAccountsToBaseAccounts(ctx sdk.Context, ak authkeeper.AccountKeeper, ek *evmkeeper.Keeper) {
	ak.IterateAccounts(ctx, func(account sdk.AccountI) (stop bool) {
		ethAcc, ok := account.(*evmostypes.EthAccount)
		if !ok {
			return false
		}

		ctx.Logger().Info(fmt.Sprintf("Migrate account %s\n", account.GetAddress().String()))

		// Migrate pubkey if pubkey type is eth_secp256k1
		legacyPubkey := ethAcc.GetPubKey()
		if legacyPubkey != nil && legacyPubkey.Type() == "eth_secp256k1" {
			pubkey := &ethsecp256k1.PubKey{
				Key: ethAcc.GetPubKey().Bytes(),
			}
			ethAcc.SetPubKey(pubkey)
		}

		// NOTE: we only need to add store entries for smart contracts
		codeHashBytes := common.HexToHash(ethAcc.CodeHash).Bytes()
		if !evmtypes.IsEmptyCodeHash(codeHashBytes) {
			// get evm mapped address
			var (
				evmAddress common.Address
				err        error
			)
			address, err := ek.GetEvmAddressMapping(ctx, account.GetAddress())
			if err != nil {
				evmAddress = common.BytesToAddress(account.GetAddress().Bytes())
				ctx.Logger().Info(fmt.Sprintf("Migrate un-mapped address %s - %s", account.GetAddress().String(), evmAddress.Hex()))
			} else {
				evmAddress = *address
				ctx.Logger().Info(fmt.Sprintf("Migrate mapped address %s - %s", account.GetAddress().String(), address.Hex()))
			}
			ek.SetCodeHash(ctx, evmAddress.Bytes(), codeHashBytes)
		}

		// Set the base account in the account keeper instead of the EthAccount
		ak.SetAccount(ctx, ethAcc.BaseAccount)

		return false
	})
}

// ReactivateStaticPrecompiles sets ActiveStaticPrecompiles param on the evm
func ActivateStaticPrecompiles(ctx sdk.Context, evmKeeper *evmkeeper.Keeper) error {
	params := evmKeeper.GetParams(ctx)
	params.ActiveStaticPrecompiles = []string{
		cmn.WasmdContractAddress,
		cmn.JsonContractAddress,
		cmn.AddrContractAddress,
		cmn.BankContractAddress,
		cmn.AuthzContractAddress,

		// TODO: Cosmos Evm precompiles?
	}
	return evmKeeper.SetParams(ctx, params)
}
