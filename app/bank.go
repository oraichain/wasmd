package app

import (
	"context"
	"sync"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	v05014 "github.com/CosmWasm/wasmd/app/upgrades/v05014"
)

var (
	sendBlacklistMu sync.RWMutex
	sendBlacklist   = map[string]struct{}{}
)

func init() {
	for _, addr := range v05014.BlacklistAddresses {
		if addr == "" {
			continue
		}
		sendBlacklist[addr] = struct{}{}
	}
}

// AddSendBlacklistAddress marks an account as blocked for outbound bank sends.
func AddSendBlacklistAddress(addr string) {
	if addr == "" {
		return
	}
	sendBlacklistMu.Lock()
	defer sendBlacklistMu.Unlock()
	sendBlacklist[addr] = struct{}{}
}

// ActivateSendBlacklist loads all fork blacklist addresses into the send restriction set.
// Enforcement still only applies when BlockHeight > ForkHeight (see BlacklistSendRestriction).
func ActivateSendBlacklist() {
	sendBlacklistMu.Lock()
	defer sendBlacklistMu.Unlock()
	for _, addr := range v05014.BlacklistAddresses {
		if addr == "" {
			continue
		}
		sendBlacklist[addr] = struct{}{}
	}
}

// IsSendBlacklisted reports whether addr is on the bank send blacklist set.
func IsSendBlacklisted(addr string) bool {
	sendBlacklistMu.RLock()
	defer sendBlacklistMu.RUnlock()
	_, ok := sendBlacklist[addr]
	return ok
}

// RegisterBankSendRestrictions wires the blacklist SendRestrictionFn onto the bank keeper.
func RegisterBankSendRestrictions(bk bankkeeper.BaseKeeper) {
	bk.AppendSendRestriction(BlacklistSendRestriction)
}

// BlacklistSendRestriction blocks outbound bank sends from blacklist addresses only after
// the fork block (height > ForkHeight). Inbound sends to those addresses remain allowed
// (funds sent in are frozen / effectively lost). At ForkHeight and before, this is a no-op.
func BlacklistSendRestriction(ctx context.Context, fromAddr, toAddr sdk.AccAddress, _ sdk.Coins) (sdk.AccAddress, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if sdkCtx.BlockHeight() <= v05014.ForkHeight {
		return toAddr, nil
	}

	sendBlacklistMu.RLock()
	defer sendBlacklistMu.RUnlock()

	if _, ok := sendBlacklist[fromAddr.String()]; ok {
		return nil, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "sender %s is blacklisted", fromAddr)
	}
	return toAddr, nil
}
