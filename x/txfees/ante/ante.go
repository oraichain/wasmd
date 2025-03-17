package ante

import (
	"bytes"
	"fmt"

	errors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	txfeeskeeper "github.com/CosmWasm/wasmd/x/txfees/keeper"
	txfeestypes "github.com/CosmWasm/wasmd/x/txfees/types"
	tmstrings "github.com/cometbft/cometbft/libs/strings"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errorstypes "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

type DeductFeeDecorator struct {
	ak  AccountKeeper
	bk  BankKeeper
	fgk FeegrantKeeper
	tfk txfeeskeeper.Keeper
}

func NewDeductFeeDecorator(
	ak AccountKeeper,
	bk BankKeeper,
	fgk FeegrantKeeper,
	tfk txfeeskeeper.Keeper,
) DeductFeeDecorator {
	return DeductFeeDecorator{
		ak:  ak,
		bk:  bk,
		fgk: fgk,
		tfk: tfk,
	}
}

func (tdfd DeductFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, errors.Wrap(errorstypes.ErrTxDecode, "Tx must be a FeeTx")
	}

	// If this is genesis height, don't check the fee.
	// This is needed so that gentx's can be created without having to pay a fee.
	if ctx.BlockHeight() == 0 {
		return next(ctx, tx, simulate)
	}

	if addr := tdfd.ak.GetModuleAddress(authtypes.FeeCollectorName); addr == nil {
		return ctx, fmt.Errorf("fee collector module account (%s) has not been set", authtypes.FeeCollectorName)
	}

	feeCoins := feeTx.GetFee()
	if len(feeCoins) > 1 {
		return ctx, txfeestypes.ErrTooManyFeeCoins
	}

	// incase this is bypass msg. We will validate msg in mempool ante
	if len(feeCoins) == 0 {
		return tdfd.DeductFeeAnteHandle(ctx, tx, simulate, next, feeTx)
	}

	feeDenom := feeCoins.GetDenomByIndex(0)
	baseDenom, err := tdfd.tfk.GetBaseTokenDenom(ctx)
	if err != nil {
		return ctx, err
	}

	// if pay with different token
	if feeDenom != baseDenom {
		isAllowed, err := tdfd.tfk.IsTokenAllowed(ctx, feeDenom)
		if err != nil {
			return ctx, err
		}

		if !isAllowed {
			return ctx, errors.Wrapf(txfeestypes.ErrTokenAllowed, "token not allowed %s", feeDenom)
		}
	}

	return tdfd.DeductFeeAnteHandle(ctx, tx, simulate, next, feeTx)
}

// DeductFeeAnteHandle deducts the fee from fee payer or fee granter (if set) and ensure
// the fee collector module account is set
func (tdfd DeductFeeDecorator) DeductFeeAnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
	feeTx sdk.FeeTx,
) (newCtx sdk.Context, err error) {
	fee := feeTx.GetFee()
	feePayer := feeTx.FeePayer()
	feeGranter := feeTx.FeeGranter()

	deductFeesFrom := feePayer

	// if feegranter set deduct fee from feegranter account.
	// this works with only when feegrant enabled.
	if feeGranter != nil {
		if tdfd.fgk == nil {
			return ctx, errors.Wrap(errorstypes.ErrInvalidRequest, "fee grants are not enabled")
		} else if !bytes.Equal(feeGranter, feePayer) {
			err := tdfd.fgk.UseGrantedFees(ctx, feeGranter, feePayer, fee, tx.GetMsgs())
			if err != nil {
				return ctx, errors.Wrapf(err, "%s not allowed to pay fees from %s", feeGranter, feePayer)
			}
		}

		deductFeesFrom = feeGranter
	}

	deductFeesFromAcc := tdfd.ak.GetAccount(ctx, deductFeesFrom)
	if deductFeesFromAcc == nil {
		return ctx, errors.Wrapf(errorstypes.ErrUnknownAddress, "fee payer address: %s does not exist", sdk.AccAddress(deductFeesFrom).String())
	}

	// deduct the fees
	if !fee.IsZero() {
		err = DeductFees(tdfd.bk, ctx, deductFeesFrom, fee)
		if err != nil {
			return ctx, err
		}
	}

	events := sdk.Events{sdk.NewEvent(sdk.EventTypeTx,
		sdk.NewAttribute(sdk.AttributeKeyFee, fee.String()),
	)}
	ctx.EventManager().EmitEvents(events)

	return next(ctx, tx, simulate)
}

// DeductFees deducts fees from the given account.
func DeductFees(bankKeeper BankKeeper, ctx sdk.Context, accAddress sdk.AccAddress, fees sdk.Coins) error {
	if err := fees.Validate(); err != nil {
		return errors.Wrapf(errorstypes.ErrInsufficientFee, "invalid fee amount: %s", fees)
	}

	if err := bankKeeper.SendCoinsFromAccountToModule(ctx, accAddress, authtypes.FeeCollectorName, fees); err != nil {
		return errors.Wrapf(errorstypes.ErrInsufficientFunds, err.Error())
	}

	return nil
}

type MempoolFeeDecorator struct {
	BypassMinFeeMsgTypes []string
	tfk                  txfeeskeeper.Keeper
}

func NewMempoolFeeDecorator(bypassMsgTypes []string, tfk txfeeskeeper.Keeper) MempoolFeeDecorator {
	return MempoolFeeDecorator{
		BypassMinFeeMsgTypes: bypassMsgTypes,
		tfk:                  tfk,
	}
}

func (mpfd MempoolFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, errors.Wrap(errorstypes.ErrTxDecode, "Tx must be a FeeTx")
	}

	// Ensure that the provided fees meet a minimum threshold for the validator,
	// if this is a CheckTx. This is only for local mempool purposes, and thus
	// is only ran on check tx.
	if !ctx.IsCheckTx() || simulate {
		return next(ctx, tx, simulate)
	}

	if ctx.BlockHeight() == 0 {
		return next(ctx, tx, simulate)
	}

	if mpfd.ContainsOnlyBypassMinFeeMsgs(feeTx.GetMsgs()) {
		return next(ctx, tx, simulate)
	}

	gas := feeTx.GetGas()
	requiredBaseFee, err := mpfd.getBaseRequiredFees(ctx, int64(gas))
	if err != nil {
		return ctx, err
	}

	feeCoins := feeTx.GetFee()
	if len(feeCoins) != 1 {
		return ctx, errors.Wrapf(errorstypes.ErrInsufficientFee,
			"Expected 1 fee denom attached, got %d", len(feeCoins))
	}

	convertedFee, err := mpfd.tfk.ConvertToBaseTokenFee(ctx, feeCoins[0])
	if err != nil {
		return ctx, err
	}

	if !(convertedFee.IsGTE(requiredBaseFee)) {
		return ctx, errors.Wrapf(errorstypes.ErrInsufficientFee, "insufficient fees; got: %s which converts to %s. required: %s", feeCoins[0], convertedFee, requiredBaseFee)

	}

	return next(ctx, tx, simulate)
}

func (mpfd MempoolFeeDecorator) getBaseRequiredFees(ctx sdk.Context, gasLimit int64) (sdk.Coin, error) {
	var (
		minGasPrices math.LegacyDec
		err          error
	)

	baseDenom, err := mpfd.tfk.GetBaseTokenDenom(ctx)
	if err != nil {
		return sdk.Coin{}, err
	}

	minGasPrices = ctx.MinGasPrices().AmountOf(baseDenom)
	glDec := math.LegacyNewDec(gasLimit)

	// fee = min gas prices * gas limit
	fee := minGasPrices.Mul(glDec)
	requiredFees := sdk.NewCoin(baseDenom, fee.Ceil().RoundInt())

	return requiredFees, nil
}

func (mpfd MempoolFeeDecorator) ContainsOnlyBypassMinFeeMsgs(msgs []sdk.Msg) bool {
	for _, msg := range msgs {
		if tmstrings.StringInSlice(sdk.MsgTypeURL(msg), mpfd.BypassMinFeeMsgTypes) {
			continue
		}
		return false
	}

	return true
}
