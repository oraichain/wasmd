package app

import (
	"errors"

	corestoretypes "cosmossdk.io/core/store"
	circuitkeeper "cosmossdk.io/x/circuit/keeper"
	globalfeekeeper "github.com/CosmosContracts/juno/v18/x/globalfee/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v8/modules/core/keeper"

	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmTypes "github.com/CosmWasm/wasmd/x/wasm/types"

	txfeeskeeper "github.com/CosmWasm/wasmd/x/txfees/keeper"

	storetypes "cosmossdk.io/store/types"
	evmosanteevm "github.com/cosmos/evm/ante/evm"
	feemarketkeeper "github.com/cosmos/evm/x/feemarket/keeper"
	evmkeeper "github.com/cosmos/evm/x/vm/keeper"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

// HandlerOptions extend the SDK's AnteHandler options by requiring the IBC
// channel keeper.
type HandlerOptions struct {
	ante.HandlerOptions
	AccountKeeper         evmtypes.AccountKeeper
	AuthzKeeper           *authzkeeper.Keeper
	IBCKeeper             *keeper.Keeper
	EvmKeeper             *evmkeeper.Keeper
	GlobalFeeKeeper       globalfeekeeper.Keeper
	StakingKeeper         stakingkeeper.Keeper
	FeeMarketKeeper       feemarketkeeper.Keeper
	WasmConfig            *wasmTypes.WasmConfig
	WasmKeeper            *wasmkeeper.Keeper
	ContractKeeper        *wasmkeeper.PermissionedKeeper
	TXCounterStoreService corestoretypes.KVStoreService
	TxCounterStoreKey     storetypes.StoreKey
	MaxTxGasWanted        uint64
	CircuitKeeper         *circuitkeeper.Keeper
	BankKeeper            *bankkeeper.BaseKeeper
	TxFeesKeeper          txfeeskeeper.Keeper
	DisabledAuthzMsgs     []string
	BypassMinFeeMsgTypes  []string
}

func (options *HandlerOptions) Validate() error {
	if options.AccountKeeper == nil {
		return errors.New("account keeper is required for ante builder")
	}
	if options.BankKeeper == nil {
		return errors.New("bank keeper is required for ante builder")
	}
	if options.SignModeHandler == nil {
		return errors.New("sign mode handler is required for ante builder")
	}
	if options.WasmConfig == nil {
		return errors.New("wasm config is required for ante builder")
	}
	if options.TXCounterStoreService == nil {
		return errors.New("wasm store service is required for ante builder")
	}
	if options.CircuitKeeper == nil {
		return errors.New("circuit keeper is required for ante builder")
	}
	if options.EvmKeeper == nil {
		return errors.New("evm keeper is required for ante builder")
	}
	if options.WasmKeeper == nil {
		return errors.New("wasm keeper is required for ante builder")
	}
	if options.ContractKeeper == nil {
		return errors.New("contract keeper is required for ante builder")
	}

	return nil
}

// newEthAnteHandler creates the default ante handler for Ethereum transactions
func newEthAnteHandler(options HandlerOptions) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		evmosanteevm.NewEVMMonoDecorator(options.AccountKeeper, options.FeeMarketKeeper, options.EvmKeeper, options.MaxTxGasWanted), // outermost AnteDecorator. SetUpContext must be called first
	)
}

// newCosmosAnteHandler creates the default ante handler for Cosmos transactions
func newCosmosAnteHandler(options HandlerOptions) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		evmosantecosmos.RejectMessagesDecorator{}, // reject MsgEthereumTxs
		NewAuthzLimiterDecorator( // disable the Msg types that cannot be included on an authz.MsgExec msgs field
			sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{}),
			sdk.MsgTypeURL(&sdkvesting.MsgCreateVestingAccount{}),
		),
		ante.NewSetUpContextDecorator(),
		ante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		evmosantecosmos.NewMinGasPriceDecorator(options.FeeMarketKeeper, options.EvmKeeper),
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		ante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, options.TxFeeChecker),
		// SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewSetPubKeyDecorator(options.AccountKeeper),
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		ante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(options.IBCKeeper),
		evmosanteevm.NewGasWantedDecorator(options.EvmKeeper, options.FeeMarketKeeper),
	)
}
