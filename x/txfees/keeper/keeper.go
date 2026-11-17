package keeper

import (
	"fmt"

	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/CosmWasm/wasmd/x/txfees/types"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Keeper struct {
	cdc          codec.BinaryCodec
	storeService storetypes.KVStoreService
	wk           *wasmkeeper.Keeper
	bankKeeper   types.BankKeeper
	accKeeper    types.AccountKeeper
	// the address capable of executing a MsgUpdateParams message. Typically, this
	// should be the x/gov module account.
	authority string
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	wk *wasmkeeper.Keeper,
	bankKeeper types.BankKeeper,
	accKeeper types.AccountKeeper,
	authority string,
) Keeper {
	// accKeeper backs assertBlacklistable. A nil one would silently turn the
	// module-account guard off and make e.g. fee_collector blacklistable, which
	// halts the chain unrecoverably. Refuse to wire that up at all.
	if accKeeper == nil {
		panic("txfees keeper: nil AccountKeeper")
	}

	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		wk:           wk,
		bankKeeper:   bankKeeper,
		accKeeper:    accKeeper,
		authority:    authority,
	}
}

// GetAuthority returns the x/txfees module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
