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
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	"github.com/cosmos/cosmos-sdk/codec"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmTypes "github.com/CosmWasm/wasmd/x/wasm/types"

	txfeesante "github.com/CosmWasm/wasmd/x/txfees/ante"
	txfeeskeeper "github.com/CosmWasm/wasmd/x/txfees/keeper"

	storetypes "cosmossdk.io/store/types"
	globalfeeante "github.com/CosmosContracts/juno/v18/x/globalfee/ante"
)

// HandlerOptions extend the SDK's AnteHandler options by requiring the IBC
// channel keeper.
// EVM ante wiring is intentionally omitted: ante does not own module stores, so
// removing EVM execution/decorators here does not affect Multistore appHash from
// keeping evm/feemarket/erc20/precisebank modules mounted.
type HandlerOptions struct {
	ante.HandlerOptions

	AccountKeeper         authkeeper.AccountKeeper
	AuthzKeeper           *authzkeeper.Keeper
	IBCKeeper             *keeper.Keeper
	GlobalFeeKeeper       globalfeekeeper.Keeper
	StakingKeeper         stakingkeeper.Keeper
	WasmConfig            *wasmTypes.WasmConfig
	WasmKeeper            *wasmkeeper.Keeper
	ContractKeeper        *wasmkeeper.PermissionedKeeper
	TXCounterStoreService corestoretypes.KVStoreService
	TxCounterStoreKey     storetypes.StoreKey
	CircuitKeeper         *circuitkeeper.Keeper
	BankKeeper            *bankkeeper.BaseKeeper
	TxFeesKeeper          txfeeskeeper.Keeper
	Codec                 codec.Codec
	DisabledAuthzMsgs     []string
	BypassMinFeeMsgTypes  []string
}

func (options *HandlerOptions) Validate() error {
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
	if options.WasmKeeper == nil {
		return errors.New("wasm keeper is required for ante builder")
	}
	if options.ContractKeeper == nil {
		return errors.New("contract keeper is required for ante builder")
	}
	if options.Codec == nil {
		return errors.New("codec is required for ante builder")
	}

	return nil
}

// newCosmosAnteHandler creates the CosmWasm/IBC ante chain without EVM decorators.
func newCosmosAnteHandler(options HandlerOptions) sdk.AnteHandler {
	sigGasConsumer := options.SigGasConsumer
	if sigGasConsumer == nil {
		sigGasConsumer = ante.DefaultSigVerificationGasConsumer
	}

	decorators := []sdk.AnteDecorator{
		ante.NewSetUpContextDecorator(),
		wasmkeeper.NewLimitSimulationGasDecorator(options.WasmConfig.SimulationGasLimit),
		wasmkeeper.NewCountTXDecorator(options.TXCounterStoreService),
		wasmkeeper.NewGasRegisterDecorator(options.WasmKeeper.GetGasRegister()),
		circuitante.NewCircuitBreakerDecorator(options.CircuitKeeper),
		ante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		wasmkeeper.NewConsumeGasForTxSizeDecorator(options.AccountKeeper, options.WasmKeeper),
		txfeesante.NewBlacklistDecorator(options.TxFeesKeeper, options.Codec),
		globalfeeante.NewFeeDecorator(options.BypassMinFeeMsgTypes, options.GlobalFeeKeeper, options.StakingKeeper, maxBypassMinFeeMsgGasUsage),
		txfeesante.NewMempoolFeeDecorator(options.BypassMinFeeMsgTypes, options.TxFeesKeeper),
		txfeesante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, options.TxFeesKeeper),
		ante.NewSetPubKeyDecorator(options.AccountKeeper),
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, sigGasConsumer),
		ante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(options.IBCKeeper),
	}

	return sdk.ChainAnteDecorators(decorators...)
}
