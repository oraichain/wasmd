package app

import (
	"errors"

	corestoretypes "cosmossdk.io/core/store"
	circuitante "cosmossdk.io/x/circuit/ante"
	circuitkeeper "cosmossdk.io/x/circuit/keeper"
	globalfeekeeper "github.com/CosmosContracts/juno/v18/x/globalfee/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibcante "github.com/cosmos/ibc-go/v8/modules/core/ante"
	"github.com/cosmos/ibc-go/v8/modules/core/keeper"

	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmTypes "github.com/CosmWasm/wasmd/x/wasm/types"

	txfeesante "github.com/CosmWasm/wasmd/x/txfees/ante"
	txfeeskeeper "github.com/CosmWasm/wasmd/x/txfees/keeper"

	storetypes "cosmossdk.io/store/types"
	globalfeeante "github.com/CosmosContracts/juno/v18/x/globalfee/ante"
	evmantecosmos "github.com/cosmos/evm/ante/cosmos"
	evmanteevm "github.com/cosmos/evm/ante/evm"
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
		evmanteevm.NewEVMMonoDecorator(options.AccountKeeper, options.FeeMarketKeeper, options.EvmKeeper, options.MaxTxGasWanted), // outermost AnteDecorator. SetUpContext must be called first
	)
}

// newCosmosAnteHandler creates the default ante handler for Cosmos transactions
func newCosmosAnteHandler(options HandlerOptions) sdk.AnteHandler {
	var sigGasConsumer = options.SigGasConsumer
	if sigGasConsumer == nil {
		sigGasConsumer = ante.DefaultSigVerificationGasConsumer
	}

	decorators := []sdk.AnteDecorator{
		evmantecosmos.RejectMessagesDecorator{},                                          // reject MsgEthereumTxs
		ante.NewSetUpContextDecorator(),                                                  // outermost AnteDecorator. SetUpContext must be called first
		wasmkeeper.NewLimitSimulationGasDecorator(options.WasmConfig.SimulationGasLimit), // after setup context to enforce limits early
		wasmkeeper.NewCountTXDecorator(options.TXCounterStoreService),
		wasmkeeper.NewGasRegisterDecorator(options.WasmKeeper.GetGasRegister()),
		circuitante.NewCircuitBreakerDecorator(options.CircuitKeeper),
		ante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		// ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		// replace by custom gas tx size for gasless contract
		wasmkeeper.NewConsumeGasForTxSizeDecorator(options.AccountKeeper, options.WasmKeeper),
		// nil so that it only checks with the min gas price of the chain, not the custom fee checker. For cosmos messages, the default tx fee checker is enough
		globalfeeante.NewFeeDecorator(options.BypassMinFeeMsgTypes, options.GlobalFeeKeeper, options.StakingKeeper, maxBypassMinFeeMsgGasUsage),
		// ante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, nil),
		txfeesante.NewMempoolFeeDecorator(options.BypassMinFeeMsgTypes, options.TxFeesKeeper),
		txfeesante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, options.TxFeesKeeper),
		// we use evmante.NewSetPubKeyDecorator so that for eth_secp256k1 accs, we can validate the signer using the evm-cosmos mapping logic
		evmantecosmos.NewSetPubKeyDecorator(options.AccountKeeper, options.EvmKeeper), // SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		ante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(options.IBCKeeper),
	}

	return sdk.ChainAnteDecorators(decorators...)
}
