package v10

import (
	"github.com/CosmWasm/wasmd/app/upgrades"
)

// Last executed block on the v9 code was 4713064.
// Last committed block is assumed to be 4713064, as we have block proposals that were not precommitted upon
// for 4713065.
const ForkHeight = 118018800 // stop at 118018794 -> run for at 118018800

// UpgradeName defines the on-chain upgrade name for the Osmosis v9-fork for recovery.
// This is not called v10, due to this bug that would require a state migration to fix:
const UpgradeName = "v0.50.14"

var Fork = upgrades.Fork{
	UpgradeName:    UpgradeName,
	UpgradeHeight:  ForkHeight,
	BeginForkLogic: RunForkLogic,
}
