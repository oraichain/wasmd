package v10

import (
	"strconv"

	sdkmath "cosmossdk.io/math"
)

// RevertEntry pairs a wallet with the ORAI amount (base units) to restore at fork.
type RevertEntry struct {
	Address string
	Amount  sdkmath.Int
}

// ForkHeightStr can be overridden at build time via ldflags, e.g.
//
//	-X github.com/CosmWasm/wasmd/app/upgrades/v05014.ForkHeightStr=40
//
// Production default remains 118018795 (halt 118018794 + 1).
var (
	ForkHeightStr string = "118018795"

	BlacklistAddresses []string = []string{
		"orai1vyghw3r3567y2algruuflqw2hx05vt6k945wrq",
		"orai1hru4a5w0c29wr36l2dgaymqqd4h0vju9tlvk8w",
		"orai1ycryq0mghafwfwr346d5qe2ce8zvmflns08khy",
	}

	// RevertAddress: each entry has Address + Amount (math.Int, orai base units).
	// Fill Amount before enabling revert mint/send in fork logic.
	RevertAddress []RevertEntry = []RevertEntry{
		{Address: "orai1cm9xkysr58kgcvzjs0shlgma557hsu62s76f98", Amount: sdkmath.NewInt(38_203_075_000_000)},
		{Address: "orai1psee0upxws59g0tct0a87ka7yp64ft9jqnm8nh", Amount: sdkmath.NewInt(8_294_510_000_000)},
		{Address: "orai15gjs0evtchjcv0vuzysgg6cd359yv9hnsej5x4", Amount: sdkmath.NewInt(1_717_788_000_000)},
		// oraidex pools
		{Address: "orai15aunrryk5yqsrgy0tvzpj7pupu62s0t2n09t0dscjgzaa27e44esefzgf8", Amount: sdkmath.NewInt(10_755_035_000_000)},
		{Address: "orai1xdl93lj4kupt45wvsrwsghgd2tu87jnyyq8ht2szu9k7cjxh2yhssd0jlu", Amount: sdkmath.NewInt(111_682_000_000)},
		{Address: "orai157ffw5c4cfcchmumwa9rd80dlpqqwzezkerpfnf2895av6nq5pyqm24mut", Amount: sdkmath.NewInt(51_489_000_000)},
		{Address: "orai1jf74ry4m0jcy9emsaudkhe7vte9l8qy8enakvs", Amount: sdkmath.NewInt(30_674_000_000)},
		{Address: "orai1037mwrt2wqgzkw05fym0jh8jh7qvdj6hejuyutfn4z0wdth5m63sy2xal3", Amount: sdkmath.NewInt(4_278_000_000)},
		// orchai
		{Address: "orai1ahug8tgnw7k78dj5567n0cch9l86ntsdtwv8z3sdd3g4jzqpg7qqqmr9fl", Amount: sdkmath.NewInt(59_255_027_000_000)},
		// obridge
		{Address: "orai195269awwnt5m6c843q6w7hp8rt0k7syfu9de4h0wz384slshuzps8y7ccm", Amount: sdkmath.NewInt(77_141_000_000)},
		// operation
		{Address: "orai1c32pwl0atr9qctffr29f9yjnnvgdu4lpmlgzxh", Amount: sdkmath.NewInt(88_436_000_000)},
		{Address: "orai1sukkexujpz2trec6x8t6c3dlngc52c2t5m2q29", Amount: sdkmath.NewInt(342_292_000_000)},
	}

	// RecoveryAssets are cw20-base contract addresses to rescue at fork.
	// Localfork / unit-test values are Instantiate2-predictable (fixMsg=false); see
	// upgrades_test.go for deployer mnemonic, salts, and address derivation.
	// Replace with mainnet contract addrs for production. Empty slice = skip CW20 rescue.
	RecoveryAssets []string = []string{
		// e2e test
		// "orai18m6m3uy7zknru9tx5jfpadcjc80jp38ngpqwrralm76kj7fgx80q7hky0y", // salt localfork-cw20-v1
		// "orai10438e2jljstgk2qwuzjzf67sd4al0v7m0lcuskdl4wlhwqmtv73s69lnft", // salt localfork-cw20-v2

		// // mainnet
		"orai15un8msx3n5zf9ahlxmfeqd2kwa5wm0nrpxer304m9nd5q6qq0g6sku5pdd",
		"orai12hzjxfh77wl572gdzct2fxv2arxcwh6gykc7qh",
		"orai1065qe48g7aemju045aeyprflytemx7kecxkf5m7u5h5mphd0qlcs47pclp",
	}

	// RecoveryFromAddress are wallets whose CW20 balances on RecoveryAssets are
	// transferred to RecoveryAddress at fork.
	RecoveryFromAddress []string = []string{
		// e2e test address
		// "orai1hru4a5w0c29wr36l2dgaymqqd4h0vju9tlvk8w",

		// mainnet address
		"orai1vyghw3r3567y2algruuflqw2hx05vt6k945wrq",
	}

	PausePool []string = []string{
		"orai1jf74ry4m0jcy9emsaudkhe7vte9l8qy8enakvs",
		"orai10s0c75gw5y5eftms5ncfknw6lzmx0dyhedn75uz793m8zwz4g8zq4d9x9a",
	}

	// RecoveryAddress receives rescued CW20. Override via ldflags for localfork:
	//
	//	-X github.com/CosmWasm/wasmd/app/upgrades/v05014.RecoveryAddress=orai1...
	RecoveryAddress string = "orai1g5yvpy7q99acamd8chsmsucpnjcxshczt5me4p"
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
