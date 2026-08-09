package v10

import (
	"github.com/CosmWasm/wasmd/app/upgrades"
)

var Fork = upgrades.Fork{
	UpgradeName:    UpgradeName,
	UpgradeHeight:  ForkHeight,
	BeginForkLogic: RunForkLogic,
}
