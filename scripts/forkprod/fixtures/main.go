// Command fixtures dumps the compiled v0.50.14 production fork fixtures as JSON so the
// e2e genesis is built from the same constants the shipping binary uses. Building the
// genesis from a hand-copied list would let the two drift silently.
//
// Must be run WITHOUT the localfork build tag.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	v05014 "github.com/CosmWasm/wasmd/app/upgrades/v05014"
)

type revert struct {
	Address string `json:"address"`
	Amount  string `json:"amount"`
}

type fixtures struct {
	ForkHeight           int64    `json:"fork_height"`
	Denom                string   `json:"denom"`
	MainnetChainID       string   `json:"mainnet_chain_id"`
	IsLocalForkBuild     bool     `json:"is_localfork_build"`
	BlacklistAddresses   []string `json:"blacklist_addresses"`
	RevertAddress        []revert `json:"revert_addresses"`
	RecoveryAssets       []string `json:"recovery_assets"`
	RecoveryNativeDenoms []string `json:"recovery_native_denoms"`
	RecoveryFromAddress  []string `json:"recovery_from_address"`
	RecoveryAddress      string   `json:"recovery_address"`
	AdminContract        string   `json:"admin_contract"`
	PausePoolV2          string   `json:"pause_pool_v2"`
	PausePoolV3          string   `json:"pause_pool_v3"`
}

func main() {
	if v05014.IsLocalForkBuild {
		fmt.Fprintln(os.Stderr, "ERROR: built with -tags localfork; forkprod needs production fixtures")
		os.Exit(1)
	}

	out := fixtures{
		ForkHeight:           v05014.ForkHeight,
		Denom:                v05014.CoinDenom,
		MainnetChainID:       v05014.MainnetChainID,
		IsLocalForkBuild:     v05014.IsLocalForkBuild,
		BlacklistAddresses:   v05014.BlacklistAddresses,
		RecoveryAssets:       v05014.RecoveryAssets,
		RecoveryNativeDenoms: v05014.RecoveryNativeDenoms,
		RecoveryFromAddress:  v05014.RecoveryFromAddress,
		RecoveryAddress:      v05014.RecoveryAddress,
		AdminContract:        v05014.AdminContract,
		PausePoolV2:          v05014.PausePoolV2,
		PausePoolV3:          v05014.PausePoolV3,
	}
	for _, r := range v05014.RevertAddress {
		out.RevertAddress = append(out.RevertAddress, revert{Address: r.Address, Amount: r.Amount.String()})
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
