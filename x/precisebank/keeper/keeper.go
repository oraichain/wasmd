package keeper

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"

	evmtypes "github.com/cosmos/evm/x/evm/types"

	"github.com/CosmWasm/wasmd/x/precisebank/types"
)

// Enforce that Keeper implements the expected keeper interfaces
var _ evmtypes.BankKeeper = Keeper{}

// Keeper defines the precisebank module's keeper
type Keeper struct {
	cdc      codec.BinaryCodec
	storeKey storetypes.StoreKey

	bk types.BankKeeper
	ak types.AccountKeeper
}

// NewKeeper creates a new keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	bk types.BankKeeper,
	ak types.AccountKeeper,
) Keeper {
	return Keeper{
		cdc:      cdc,
		storeKey: storeKey,
		bk:       bk,
		ak:       ak,
	}
}
