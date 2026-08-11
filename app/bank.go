package app

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	v05014 "github.com/CosmWasm/wasmd/app/upgrades/v05014"
	txfeeskeeper "github.com/CosmWasm/wasmd/x/txfees/keeper"
)

// RegisterBankSendRestrictions blocks outbound bank sends from blacklisted accounts once
// the fork block has passed (height > ForkHeight). Inbound sends to those accounts remain
// allowed (funds sent in are frozen). At ForkHeight and before this is a no-op, so the fork
// logic itself can still move funds out of blacklisted accounts.
//
// The blacklist is read from the txfees store — the same source x/txfees ante
// BlacklistDecorator uses — so there is a single source of truth and a gov-added entry
// takes effect on both the ante and the bank paths.
func RegisterBankSendRestrictions(bk bankkeeper.BaseKeeper, tfk txfeeskeeper.Keeper) {
	bk.AppendSendRestriction(func(ctx context.Context, fromAddr, toAddr sdk.AccAddress, _ sdk.Coins) (sdk.AccAddress, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		if sdkCtx.BlockHeight() <= v05014.ForkHeight {
			return toAddr, nil
		}
		blacklisted, err := tfk.IsBlacklisted(sdkCtx, fromAddr)
		if err != nil {
			return nil, err
		}
		if blacklisted {
			return nil, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "sender %s is blacklisted", fromAddr)
		}
		return toAddr, nil
	})
}
