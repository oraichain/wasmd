package v10

import (
	"strconv"
)

// ForkHeightStr can be overridden at build time via ldflags, e.g.
//
//	-X github.com/CosmWasm/wasmd/app/upgrades/v05014.ForkHeightStr=40
//
// Production default remains 118018795 (halt 118018794 + 1).
var (
	ForkHeightStr string = "118018795"

	BlacklistAddresses []string = []string{
		"orai1hru4a5w0c29wr36l2dgaymqqd4h0vju9tlvk8w",
		"orai1ycryq0mghafwfwr346d5qe2ce8zvmflns08khy",
		"orai1ahug8tgnw7k78dj5567n0cch9l86ntsdtwv8z3sdd3g4jzqpg7qqqmr9fl",
		"orai1cm9xkysr58kgcvzjs0shlgma557hsu62s76f98",
		"orai15aunrryk5yqsrgy0tvzpj7pupu62s0t2n09t0dscjgzaa27e44esefzgf8",
	}

	// RecoveryAssets are cw20-base contract addresses to rescue at fork.
	// Localfork / unit-test values are Instantiate2-predictable (fixMsg=false); see
	// upgrades_test.go for deployer mnemonic, salts, and address derivation.
	// Replace with mainnet contract addrs for production. Empty slice = skip CW20 rescue.
	RecoveryAssets []string = []string{
		// e2e test contracts
		"orai18m6m3uy7zknru9tx5jfpadcjc80jp38ngpqwrralm76kj7fgx80q7hky0y", // salt localfork-cw20-v1
		"orai10438e2jljstgk2qwuzjzf67sd4al0v7m0lcuskdl4wlhwqmtv73s69lnft", // salt localfork-cw20-v2

		// // mainnet contracts
		// "orai15un8msx3n5zf9ahlxmfeqd2kwa5wm0nrpxer304m9nd5q6qq0g6sku5pdd",
		// "orai12hzjxfh77wl572gdzct2fxv2arxcwh6gykc7qh",
		// "orai1065qe48g7aemju045aeyprflytemx7kecxkf5m7u5h5mphd0qlcs47pclp",
	}

	// RecoveryFromAddress are wallets whose CW20 balances on RecoveryAssets are
	// transferred to RecoveryAddress at fork.
	RecoveryFromAddress []string = []string{
		// e2e test address
		"orai1hru4a5w0c29wr36l2dgaymqqd4h0vju9tlvk8w",

		// mainnet address
	}

	// RecoveryAddress receives rescued CW20. Override via ldflags for localfork:
	//
	//	-X github.com/CosmWasm/wasmd/app/upgrades/v05014.RecoveryAddress=orai1...
	RecoveryAddress string = ""
)

// ForkHeight is the block height at which RunForkLogic executes.
var ForkHeight = mustParseForkHeight(ForkHeightStr)

// UpgradeName defines the on-chain upgrade name for the v0.50.14 hard fork.
const UpgradeName = "v0.50.14"

func mustParseForkHeight(s string) int64 {
	h, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		panic("invalid ForkHeightStr: " + s)
	}
	return h
}
