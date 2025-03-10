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
	// the address capable of executing a MsgUpdateParams message. Typically, this
	// should be the x/gov module account.
	authority string
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	wk *wasmkeeper.Keeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		wk:           wk,
		authority:    authority,
	}
}

func (Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
