package ante

import (
	"bytes"
	"fmt"

	errors "cosmossdk.io/errors"
	txfeeskeeper "github.com/CosmWasm/wasmd/x/txfees/keeper"
	txfeestypes "github.com/CosmWasm/wasmd/x/txfees/types"
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

func NewTxFeesDeductFeeDecorator(
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

	feeCoins := feeTx.GetFee()

	if len(feeCoins) > 1 {
		return ctx, txfeestypes.ErrTooManyFeeCoins
	}

	if addr := tdfd.ak.GetModuleAddress(authtypes.FeeCollectorName); addr == nil {
		return ctx, fmt.Errorf("fee collector module account (%s) has not been set", authtypes.FeeCollectorName)
	}

	fee := feeTx.GetFee()
	if len(fee) == 0 {
		return tdfd.normalDeductFeeAnteHandle(ctx, tx, simulate, next, feeTx)
	}

	feeDenom := feeCoins.GetDenomByIndex(0)
	isAllowed, err := tdfd.tfk.IsTokenAllowed(ctx, feeDenom)
	if err != nil {
		return ctx, err
	}

	if !isAllowed {
		return tdfd.normalDeductFeeAnteHandle(ctx, tx, simulate, next, feeTx)
	}

	return tdfd.dynamicDeductFeeAnteHandle(ctx, tx, simulate, next, feeTx, feeDenom)
}

// normalDeductFeeAnteHandle deducts the fee from fee payer or fee granter (if set) and ensure
// the fee collector module account is set
func (tdfd DeductFeeDecorator) normalDeductFeeAnteHandle(
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
		return ctx, errors.Wrapf(errorstypes.ErrUnknownAddress, "fee payer address: %s does not exist", deductFeesFrom)
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

func (tdfd DeductFeeDecorator) dynamicDeductFeeAnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
	feeTx sdk.FeeTx,
	denom string,
) (newCtx sdk.Context, err error) {
	config, found := tdfd.tfk.GetTokenConfiguration(ctx, denom)
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

type MempoolFeeDecorator struct{}
