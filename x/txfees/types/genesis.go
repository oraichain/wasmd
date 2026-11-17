package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
	}
}

func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(gs.BlacklistAddresses))
	for _, addr := range gs.BlacklistAddresses {
		if _, err := sdk.AccAddressFromBech32(addr); err != nil {
			return fmt.Errorf("invalid blacklist address %q: %w", addr, err)
		}
		if _, dup := seen[addr]; dup {
			return fmt.Errorf("duplicate blacklist address %q", addr)
		}
		seen[addr] = struct{}{}

		// Mirrors the first check in the msg server's assertBlacklistable:
		// freezing the foundation policy would freeze the only account able to
		// unfreeze anything. Module accounts cannot be checked here (no keeper),
		// so InitGenesis stays trusted input for those.
		if IsBlacklistAuthority(addr) {
			return fmt.Errorf("blacklist address %q is the foundation x/group policy account", addr)
		}
	}

	return nil
}
