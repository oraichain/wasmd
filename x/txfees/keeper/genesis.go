package keeper

import (
	"fmt"

	"github.com/CosmWasm/wasmd/x/txfees/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the txfees module's state from a provided genesis
// state.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	if err := k.SetParams(ctx, genState.Params); err != nil {
		panic(fmt.Errorf("txfees: init genesis params: %w", err))
	}

	// Restore the blacklist. Addresses were already validated by
	// GenesisState.Validate, but panic rather than skip on any error: starting a
	// chain with a partially restored blacklist would unfreeze accounts that are
	// meant to stay frozen.
	for _, raw := range genState.BlacklistAddresses {
		addr, err := sdk.AccAddressFromBech32(raw)
		if err != nil {
			panic(fmt.Errorf("txfees: init genesis blacklist address %q: %w", raw, err))
		}
		if err := k.AddBlacklist(ctx, addr); err != nil {
			panic(fmt.Errorf("txfees: init genesis add blacklist %s: %w", raw, err))
		}
	}
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	params, err := k.GetParams(ctx)
	if err != nil {
		panic(fmt.Errorf("txfees: export genesis params: %w", err))
	}

	// Iteration is over a prefix iterator, so the exported order is the store's
	// byte order and the export is reproducible.
	var blacklist []string
	k.IterateBlacklist(ctx, func(addr sdk.AccAddress) bool {
		blacklist = append(blacklist, addr.String())
		return false
	})

	return &types.GenesisState{
		Params:             params,
		BlacklistAddresses: blacklist,
	}
}
