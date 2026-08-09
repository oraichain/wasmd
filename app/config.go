package app

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

const (
	// Mainnet
	OraichainID = "Oraichain"
	Denom       = "orai"

	// Testnet
	OraichainTestnetId = "Oraichain-testnet"

	// Interchaintest
	IctOraichainID = "orai-1"

	// Local testnet
	localnetChainID = "testing"
)

// EVMOptionsFn defines a function type for setting app options specifically for
// the Cosmos EVM app. The function should receive the chainID and return an error if
// any.
type EVMOptionsFn func(string) error

// NoOpEVMOptions is a no-op function that can be used when the app does not
// need any specific configuration.
func NoOpEVMOptions(_ string) error {
	return nil
}

var sealed = false

// ChainsCoinInfo is a map of the chain id and its corresponding EvmCoinInfo
// that allows initializing the app with different coin info based on the
// chain id
var ChainsCoinInfo = map[string]evmtypes.EvmCoinInfo{
	OraichainID: {
		Denom: Denom,
		// DisplayDenom: DisplayDenom,
		Decimals: evmtypes.SixDecimals,
	},
	OraichainTestnetId: {
		Denom: Denom,
		// DisplayDenom: DisplayDenom,
		Decimals: evmtypes.SixDecimals,
	},
	IctOraichainID: {
		Denom: Denom,
		// DisplayDenom: DisplayDenom,
		Decimals: evmtypes.SixDecimals,
	},
	localnetChainID: {
		Denom: Denom,
		// DisplayDenom: DisplayDenom,
		Decimals: evmtypes.SixDecimals,
	},
}

// EvmAppOptions registers the base denom for the chain.
// EVMConfigurator / execution config is omitted (EVM AppModules soft-removed).
func EvmAppOptions(chainID string) error {
	if sealed {
		return nil
	}

	coinInfo, found := ChainsCoinInfo[chainID]
	if !found {
		return fmt.Errorf("unknown chain id: %s", chainID)
	}

	if err := setBaseDenom(coinInfo); err != nil {
		return err
	}

	sealed = true
	return nil
}

// setBaseDenom registers the display denom and base denom and sets the
// base denom for the chain.
func setBaseDenom(ci evmtypes.EvmCoinInfo) error {
	// if err := sdk.RegisterDenom(ci.DisplayDenom, math.LegacyOneDec()); err != nil {
	// 	return err
	// }

	// // sdk.RegisterDenom will automatically overwrite the base denom when the
	// // new setBaseDenom() are lower than the current base denom's units.
	return sdk.RegisterDenom(ci.Denom, math.LegacyNewDecWithPrec(1, int64(ci.Decimals)))
}
