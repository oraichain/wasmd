package v10

import (
	"strconv"
)

// forkHeightStr can be overridden at build time via ldflags, e.g.
//
//	-X github.com/CosmWasm/wasmd/app/upgrades/v05014.forkHeightStr=40
//
// Production default remains 118018795 (halt 118018794 + 1).
var forkHeightStr = "118018795"

// ForkHeight is the block height at which RunForkLogic executes.
var ForkHeight = mustParseForkHeight(forkHeightStr)

// UpgradeName defines the on-chain upgrade name for the v0.50.14 hard fork.
const UpgradeName = "v0.50.14"

func mustParseForkHeight(s string) int64 {
	h, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		panic("invalid forkHeightStr: " + s)
	}
	return h
}
