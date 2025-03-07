package keeper

import (
	"github.com/CosmWasm/wasmd/x/txfees/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the txfees module's state from a provided genesis
// state.
// TODO: write init genesis
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {

}

// TODO: write export genesis
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{}
}
