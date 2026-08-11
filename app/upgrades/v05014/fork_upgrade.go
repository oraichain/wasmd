package v05014

import (
	"github.com/CosmWasm/wasmd/app/upgrades"
)

var Fork = upgrades.Fork{
	UpgradeName:    UpgradeName,
	UpgradeHeight:  ForkHeight,
	BeginForkLogic: RunForkLogic,
}
