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
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/evm/crypto/ethsecp256k1"
)

const maxBypassMinFeeMsgGasUsage = 1_000_000

// NewAnteHandler returns the Cosmos ante handler.
// EVM txs are unsupported; any tx extension options are rejected by
// ExtensionOptionsDecorator (checker is nil).
func NewAnteHandler(options HandlerOptions) sdk.AnteHandler {
	return func(
		ctx sdk.Context, tx sdk.Tx, sim bool,
	) (newCtx sdk.Context, err error) {
		switch tx.(type) {
		case sdk.Tx:
			return newCosmosAnteHandler(options)(ctx, tx, sim)
		default:
			return ctx, errorsmod.Wrapf(errortypes.ErrUnknownRequest, "invalid transaction type: %T", tx)
		}
	}
}

const (
	secp256k1VerifyCost uint64 = 21000
)

func DefaultSigGasConsumer(
	meter storetypes.GasMeter, sig signing.SignatureV2, params authtypes.Params,
) error {
	// Keep eth_secp256k1 support for existing Cosmos accounts that use that pubkey type.
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
