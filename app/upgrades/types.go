package upgrades

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	txfeeskeeper "github.com/CosmWasm/wasmd/x/txfees/keeper"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	evmkeeper "github.com/cosmos/evm/x/vm/keeper"
	icacontrollerkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/keeper"
	ibcfeekeeper "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/keeper"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
)

type AppKeepers struct {
	AccountKeeper             *authkeeper.AccountKeeper
	BankKeeper                bankkeeper.Keeper
	ParamsKeeper              *paramskeeper.Keeper
	ConsensusParamsKeeper     *consensusparamkeeper.Keeper
	Codec                     codec.Codec
	GetStoreKey               func(storeKey string) *storetypes.KVStoreKey
	CapabilityKeeper          *capabilitykeeper.Keeper
	ScopedICAControllerKeeper *capabilitykeeper.ScopedKeeper
	ScopedIBCKeeper           *capabilitykeeper.ScopedKeeper
	GovKeeper                 *govkeeper.Keeper
	ICAControllerKeeper       icacontrollerkeeper.Keeper
	TxFeesKeeper              txfeeskeeper.Keeper
	IBCFeeKeeper              ibcfeekeeper.Keeper
	IBCKeeper                 *ibckeeper.Keeper
	MintKeeper                *mintkeeper.Keeper
	EvmKeeper                 *evmkeeper.Keeper
}
type ModuleManager interface {
	RunMigrations(ctx context.Context, cfg module.Configurator, fromVM module.VersionMap) (module.VersionMap, error)
	GetVersionMap() module.VersionMap
}

// Upgrade defines a struct containing necessary fields that a SoftwareUpgradeProposal
// must have written, in order for the state migration to go smoothly.
// An upgrade must implement this struct, and then set it in the app.go.
// The app.go will then define the handler.
type Upgrade struct {
	// Upgrade version name, for the upgrade handler, e.g. `v7`
	UpgradeName string

	// CreateUpgradeHandler defines the function that creates an upgrade handler
	CreateUpgradeHandler func(ModuleManager, module.Configurator, *AppKeepers, map[string]*storetypes.KVStoreKey, codec.BinaryCodec) upgradetypes.UpgradeHandler
	StoreUpgrades        storetypes.StoreUpgrades
}

// Fork defines a struct containing the requisite fields for a non-software upgrade proposal
// Hard Fork at a given height to implement.
// There is one time code that can be added for the start of the Fork, in `BeginForkLogic`.
// Any other change in the code should be height-gated, if the goal is to have old and new binaries
// to be compatible prior to the upgrade height.
type Fork struct {
	// Upgrade version name, for the upgrade handler, e.g. `v7`
	UpgradeName string
	// height the upgrade occurs at
	UpgradeHeight int64

	// Function that runs some custom state transition code at the beginning of a fork.
	BeginForkLogic func(ctx sdk.Context, keepers *AppKeepers)
}
