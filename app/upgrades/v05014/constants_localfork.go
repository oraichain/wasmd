//go:build localfork

package v05014

import sdkmath "cosmossdk.io/math"

// Localfork / e2e fixtures. Built with -tags localfork (see scripts/localfork/Dockerfile).
//
// RecoveryAssets match Instantiate2 (fixMsg=false) from cw20-deployer mnemonic
// salts localfork-cw20-v1 / v2 (see 02b-deploy-cw20.sh).
// RevertAddress.Amount is the burn amount; genesis funds 2x so remaining == keep
// in revert-addresses.json.
// Pause pools disabled (no oraidex contracts on mock chain).
//
// IsLocalForkBuild=true so RunForkLogic refuses Oraichain at mainnet-scale heights.
const IsLocalForkBuild = true

var (
	ForkHeightStr string = "60"
	CoinDenom     string = "orai"

	BlacklistAddresses []string = []string{
		"orai1vyghw3r3567y2algruuflqw2hx05vt6k945wrq",
		"orai1hru4a5w0c29wr36l2dgaymqqd4h0vju9tlvk8w",
		"orai1ycryq0mghafwfwr346d5qe2ce8zvmflns08khy",
	}

	// Burn amounts. Normal wallets: genesis=2×burn so remaining keep == burn (asserted by
	// 06-verify). Pool address: after burn, remainder is sent to RecoveryAddress (bal→0).
	RevertAddress []RevertEntry = []RevertEntry{
		{Address: "orai1cm9xkysr58kgcvzjs0shlgma557hsu62s76f98", Amount: sdkmath.NewInt(6_666_000_000)},
		{Address: "orai1psee0upxws59g0tct0a87ka7yp64ft9jqnm8nh", Amount: sdkmath.NewInt(2_384_000_000)},
		{Address: "orai15gjs0evtchjcv0vuzysgg6cd359yv9hnsej5x4", Amount: sdkmath.NewInt(1_372_000_000)},
		// oraidex pool — special-case remainder → RecoveryAddress (fork.go)
		{Address: "orai15aunrryk5yqsrgy0tvzpj7pupu62s0t2n09t0dscjgzaa27e44esefzgf8", Amount: sdkmath.NewInt(10_000_000_000)},
	}

	RecoveryAssets []string = []string{
		"orai18m6m3uy7zknru9tx5jfpadcjc80jp38ngpqwrralm76kj7fgx80q7hky0y", // salt localfork-cw20-v1
		"orai10438e2jljstgk2qwuzjzf67sd4al0v7m0lcuskdl4wlhwqmtv73s69lnft", // salt localfork-cw20-v2
	}

	RecoveryNativeDenoms []string = []string{
		"factory/orai1wuvhex9xqs3r539mvc6mtm7n20fcj3qr2m0y9khx6n5vtlngfzes3k0rq9/D7yP4ycfsRWUGYionGpi64sLF2ddZ2JXxuRAti2M7uck",
		"factory/orai1wuvhex9xqs3r539mvc6mtm7n20fcj3qr2m0y9khx6n5vtlngfzes3k0rq9/oraiJP7H3LAt57DkFXNLDbLdBFNRRPvS8jg2j5AZkd9",
		"factory/orai1wuvhex9xqs3r539mvc6mtm7n20fcj3qr2m0y9khx6n5vtlngfzes3k0rq9/oraix39mVDGnusyjag97Tz5H8GvGriSZmhVvkvXRoc4",
	}

	// CW20 + native rescue source (also in BlacklistAddresses; minted by 02b/02c).
	RecoveryFromAddress []string = []string{
		"orai1hru4a5w0c29wr36l2dgaymqqd4h0vju9tlvk8w",
	}

	AdminContract string = ""
	PausePoolV2   string = ""
	PausePoolV3   string = ""

	// Overridden via ldflags to tester address after 02b-deploy-cw20.sh.
	RecoveryAddress string = ""
)

// ForkHeight is the block height at which RunForkLogic executes.
var ForkHeight = mustParseForkHeight(ForkHeightStr)
