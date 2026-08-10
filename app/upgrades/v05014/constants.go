//go:build !localfork

package v05014

import sdkmath "cosmossdk.io/math"

// Production / default fork fixtures (mainnet).
// Localfork e2e uses constants_localfork.go (build tag `localfork`).
//
// ForkHeightStr / RecoveryAddress can still be overridden via ldflags.
var (
	ForkHeightStr string = "118018795"
	CoinDenom     string = "orai"

	BlacklistAddresses []string = []string{
		"orai1vyghw3r3567y2algruuflqw2hx05vt6k945wrq",
		"orai1hru4a5w0c29wr36l2dgaymqqd4h0vju9tlvk8w",
		"orai1ycryq0mghafwfwr346d5qe2ce8zvmflns08khy",
	}

	// RevertAddress: Amount is the illicit ORAI delta to burn (base units).
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

	RecoveryAssets []string = []string{
		"orai15un8msx3n5zf9ahlxmfeqd2kwa5wm0nrpxer304m9nd5q6qq0g6sku5pdd",
		"orai12hzjxfh77wl572gdzct2fxv2arxcwh6gykc7qh",
		"orai1065qe48g7aemju045aeyprflytemx7kecxkf5m7u5h5mphd0qlcs47pclp",
	}

	RecoveryNativeDenoms []string = []string{
		"factory/orai1wuvhex9xqs3r539mvc6mtm7n20fcj3qr2m0y9khx6n5vtlngfzes3k0rq9/D7yP4ycfsRWUGYionGpi64sLF2ddZ2JXxuRAti2M7uck",
		"factory/orai1wuvhex9xqs3r539mvc6mtm7n20fcj3qr2m0y9khx6n5vtlngfzes3k0rq9/oraiJP7H3LAt57DkFXNLDbLdBFNRRPvS8jg2j5AZkd9",
		"factory/orai1wuvhex9xqs3r539mvc6mtm7n20fcj3qr2m0y9khx6n5vtlngfzes3k0rq9/oraix39mVDGnusyjag97Tz5H8GvGriSZmhVvkvXRoc4",
	}

	RecoveryFromAddress []string = []string{
		"orai1vyghw3r3567y2algruuflqw2hx05vt6k945wrq",
	}

	AdminContract string = "orai1wn0qfdhn7xfn7fvsx6fme96x4mcuzrm9wm3mvlunp5e737rpgt4qndmfv8"
	PausePoolV2   string = "orai1jf74ry4m0jcy9emsaudkhe7vte9l8qy8enakvs"
	PausePoolV3   string = "orai10s0c75gw5y5eftms5ncfknw6lzmx0dyhedn75uz793m8zwz4g8zq4d9x9a"

	RecoveryAddress string = "orai1g5yvpy7q99acamd8chsmsucpnjcxshczt5me4p"
)

// ForkHeight is the block height at which RunForkLogic executes.
var ForkHeight = mustParseForkHeight(ForkHeightStr)
