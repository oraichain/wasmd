package v05014

import (
	"strconv"

	sdkmath "cosmossdk.io/math"
)

// RevertEntry pairs a wallet with the ORAI amount (base units) to burn at fork.
type RevertEntry struct {
	Address string
	Amount  sdkmath.Int
}

// UpgradeName defines the on-chain upgrade name for the v0.50.14 hard fork.
const UpgradeName = "v0.50.14"

func mustParseForkHeight(s string) int64 {
	h, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		panic("invalid ForkHeightStr: " + s)
	}
	return h
}
