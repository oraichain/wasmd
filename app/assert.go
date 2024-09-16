package app

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// There are two places where the bech32 "gravity" account prefix can be set: the default sdkConfig and also NativeHrp
// in bech32ibc, this method asserts that the Gravity app has been properly configured with matching bech32 prefix
// Note: These checks are not in Gravity.ValidateMembers() because GetNativeHrp() requires a ctx, call this func
// just once on startup since sdkConfig is immutable and NativeHrp is not set by users.
func (app *WasmApp) assertBech32PrefixMatches(ctx sdk.Context) {
	config := sdk.GetConfig()
	if app == nil || config == nil || app.Bech32IbcKeeper == nil {
		panic("Invalid app/config/keeper state")
	}

	nativePrefix, err := app.Bech32IbcKeeper.GetNativeHrp(ctx)
	if err != nil {
		panic(errorsmod.Wrap(err, "Error obtaining bech32ibc NativeHrp"))
	}
	configPrefix := config.GetBech32AccountAddrPrefix()
	if nativePrefix != configPrefix {
		panic(fmt.Sprintf("Mismatched bech32ibc NativeHrp (%v) and config Bech32 Account Prefix (%v)",
			nativePrefix, configPrefix))
	}
}