package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// AccountKeeper defines the auth methods used by txfees to classify accounts.
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	// GetModulePermissions covers module accounts that have not been
	// instantiated in the auth store yet, which GetAccount cannot see.
	GetModulePermissions() map[string]authtypes.PermissionsForAddress
}

// BankKeeper defines the bank methods used by txfees for blacklist recovery.
type BankKeeper interface {
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
}
