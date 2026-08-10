package v05014

import (
	"fmt"
	"strconv"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RevertEntry pairs a wallet with the ORAI amount (base units) to burn at fork.
type RevertEntry struct {
	Address string
	Amount  sdkmath.Int
}

// UpgradeName defines the on-chain upgrade name for the v0.50.14 hard fork.
const UpgradeName = "v0.50.14"

// MainnetChainID is Oraichain production chain-id.
const MainnetChainID = "Oraichain"

// mainnetHeightFloor rejects localfork-tagged binaries near/at real mainnet heights.
// Localfork e2e may use chain-id Oraichain at low mock heights (see scripts/localfork).
const mainnetHeightFloor int64 = 1_000_000

func mustParseForkHeight(s string) int64 {
	h, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		panic("invalid ForkHeightStr: " + s)
	}
	return h
}

// assertNotLocalForkOnMainnet panics if this binary was built with -tags localfork
// and is executing fork logic on Oraichain at a mainnet-scale height.
func assertNotLocalForkOnMainnet(ctx sdk.Context) {
	if !IsLocalForkBuild {
		return
	}
	if ctx.ChainID() == MainnetChainID && ctx.BlockHeight() >= mainnetHeightFloor {
		panic(fmt.Sprintf(
			"localfork-tagged binary refused on %s at height %d (built with -tags localfork / constants_localfork.go); rebuild without localfork for mainnet",
			ctx.ChainID(), ctx.BlockHeight(),
		))
	}
}
