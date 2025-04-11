package app

import (
	"fmt"
	"runtime/debug"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"

	"github.com/cosmos/cosmos-sdk/types/tx/signing"

	storetypes "cosmossdk.io/store/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/evm/crypto/ethsecp256k1"
)

const maxBypassMinFeeMsgGasUsage = 1_000_000

// NewAnteHandler constructor
// NewAnteHandler returns an ante handler responsible for attempting to route an
// Ethereum or SDK transaction to an internal ante handler for performing
// transaction-level processing (e.g. fee payment, signature verification) before
// being passed onto it's respective handler.
func NewAnteHandler(options HandlerOptions) sdk.AnteHandler {
	return func(
		ctx sdk.Context, tx sdk.Tx, sim bool,
	) (newCtx sdk.Context, err error) {
		var anteHandler sdk.AnteHandler

		txWithExtensions, ok := tx.(authante.HasExtensionOptionsTx)
		if ok {
			opts := txWithExtensions.GetExtensionOptions()
			if len(opts) > 0 {
				switch typeURL := opts[0].GetTypeUrl(); typeURL {
				case "/cosmos.evm.vm.v1.ExtensionOptionsEthereumTx":
					// handle as *evmtypes.MsgEthereumTx
					anteHandler = newEthAnteHandler(options)
				case "/cosmos.evm.types.v1.ExtensionOptionDynamicFeeTx":
					// cosmos-sdk tx with dynamic fee extension
					anteHandler = newCosmosAnteHandler(options)
				default:
					return ctx, errorsmod.Wrapf(
						errortypes.ErrUnknownExtensionOptions,
						"rejecting tx with unsupported extension option: %s", typeURL,
					)
				}

				return anteHandler(ctx, tx, sim)
			}
		}

		// handle as totally normal Cosmos SDK tx
		switch tx.(type) {
		case sdk.Tx:
			anteHandler = newCosmosAnteHandler(options)
		default:
			return ctx, errorsmod.Wrapf(errortypes.ErrUnknownRequest, "invalid transaction type: %T", tx)
		}

		return anteHandler(ctx, tx, sim)
	}
}

// // NewAnteHandler returns an AnteHandler that checks and increments sequence
// // numbers, checks signatures & account numbers, and deducts fees from the first
// // signer.
// func newCosmosAnteHandler(options HandlerOptions) sdk.AnteHandler {

// 	var sigGasConsumer = options.SigGasConsumer
// 	if sigGasConsumer == nil {
// 		sigGasConsumer = ante.DefaultSigVerificationGasConsumer
// 	}

// 	decorators := []sdk.AnteDecorator{
// 		evmante.RejectMessagesDecorator{}, // reject MsgEthereumTxs
// 		ante.NewSetUpContextDecorator(),   // outermost AnteDecorator. SetUpContext must be called first
// 		wasmkeeper.NewLimitSimulationGasDecorator(options.WasmConfig.SimulationGasLimit), // after setup context to enforce limits early
// 		wasmkeeper.NewCountTXDecorator(options.TXCounterStoreService),
// 		wasmkeeper.NewGasRegisterDecorator(options.WasmKeeper.GetGasRegister()),
// 		circuitante.NewCircuitBreakerDecorator(options.CircuitKeeper),
// 		ante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
// 		ante.NewValidateBasicDecorator(),
// 		ante.NewTxTimeoutHeightDecorator(),
// 		ante.NewValidateMemoDecorator(options.AccountKeeper),
// 		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
// 		// nil so that it only checks with the min gas price of the chain, not the custom fee checker. For cosmos messages, the default tx fee checker is enough
// 		globalfeeante.NewFeeDecorator(options.BypassMinFeeMsgTypes, options.GlobalFeeKeeper, options.StakingKeeper, maxBypassMinFeeMsgGasUsage),
// 		// ante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, nil),
// 		txfeesante.NewMempoolFeeDecorator(options.BypassMinFeeMsgTypes, options.TxFeesKeeper),
// 		txfeesante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, options.TxFeesKeeper),
// 		// we use evmante.NewSetPubKeyDecorator so that for eth_secp256k1 accs, we can validate the signer using the evm-cosmos mapping logic
// 		evmante.NewSetPubKeyDecorator(options.AccountKeeper, options.EvmKeeper), // SetPubKeyDecorator must be called before all signature verification decorators
// 		ante.NewValidateSigCountDecorator(options.AccountKeeper),
// 		ante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
// 		ante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
// 		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
// 		ibcante.NewRedundantRelayDecorator(options.IBCKeeper),
// 	}

// 	return sdk.ChainAnteDecorators(decorators...)
// }

// func newEthAnteHandler(options HandlerOptions) sdk.AnteHandler {
// 	return sdk.ChainAnteDecorators(
// 		evmante.NewEthSetUpContextDecorator(options.EvmKeeper), // outermost AnteDecorator. SetUpContext must be called first
// 		evmante.NewEthMempoolFeeDecorator(options.EvmKeeper),   // Check eth effective gas price against minimal-gas-prices
// 		evmante.NewEthValidateBasicDecorator(options.EvmKeeper),
// 		evmante.NewEthSigVerificationDecorator(options.EvmKeeper),
// 		evmante.NewEthAccountVerificationDecorator(options.AccountKeeper, options.EvmKeeper),
// 		evmante.NewEthGasConsumeDecorator(options.EvmKeeper, options.MaxTxGasWanted),
// 		evmante.NewCanTransferDecorator(options.EvmKeeper),
// 		evmante.NewEthIncrementSenderSequenceDecorator(options.AccountKeeper, options.EvmKeeper), // innermost AnteDecorator.
// 	)
// }

const (
	secp256k1VerifyCost uint64 = 21000
)

func DefaultSigGasConsumer(
	meter storetypes.GasMeter, sig signing.SignatureV2, params authtypes.Params,
) error {
	// support for ethereum ECDSA secp256k1 keys
	_, ok := sig.PubKey.(*ethsecp256k1.PubKey)
	if ok {
		meter.ConsumeGas(secp256k1VerifyCost, "ante verify: eth_secp256k1")
		return nil
	}

	return ante.DefaultSigVerificationGasConsumer(meter, sig, params)
}

func Recover(logger log.Logger, err *error) {
	if r := recover(); r != nil {
		*err = errorsmod.Wrapf(errorsmod.ErrPanic, "%v", r)

		if e, ok := r.(error); ok {
			logger.Error(
				"ante handler panicked",
				"error", e,
				"stack trace", string(debug.Stack()),
			)
		} else {
			logger.Error(
				"ante handler panicked",
				"recover", fmt.Sprintf("%v", r),
			)
		}
	}
}
