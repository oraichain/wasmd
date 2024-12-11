package interchaintest

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"cosmossdk.io/math"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/icza/dyno"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos/wasm"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
)

const (
	votingPeriod     = "10s"
	maxDepositPeriod = "10s"

	// Chain and relayer version inf
	IBCRelayerImage     = "ghcr.io/cosmos/relayer"
	IBCRelayerVersion   = "latest"
	GaiaImageVersion    = "v14.1.0"
	OsmosisImageVersion = "v22.0.1"
)

var (
	repo, version = GetDockerImageInfo()

	oraiImage = ibc.DockerImage{
		Repository: repo,
		Version:    version,
		UidGid:     "1025:1025",
	}

	oraiConfig = ibc.ChainConfig{
		Type:                "cosmos",
		Name:                "orai",
		ChainID:             "orai-1",
		Images:              []ibc.DockerImage{oraiImage},
		Bin:                 "oraid",
		Bech32Prefix:        "orai",
		Denom:               "orai",
		CoinType:            "118",
		GasPrices:           "0.005orai",
		GasAdjustment:       1.5,
		TrustingPeriod:      "112h",
		NoHostMount:         false,
		ModifyGenesis:       modifyGenesisShortProposals(votingPeriod, maxDepositPeriod),
		ConfigFileOverrides: nil,
		EncodingConfig:      oraiEncoding(),
	}
	genesisWalletAmount = math.NewInt(100_000_000_000)
	amountToSend        = math.NewInt(1_000_000_000)

	pathOraiGaia = "orai-gaia"
)

// oraiEncoding registers the Orai specific module codecs so that the associated types and msgs
// will be supported when writing to the blocksdb sqlite database.
func oraiEncoding() *moduletestutil.TestEncodingConfig {
	cfg := wasm.WasmEncoding()

	return cfg
}

// GetDockerImageInfo returns the appropriate repo and branch version string for integration with the CI pipeline.
// The remote runner sets the BRANCH_CI env var. If present, interchaintest will use the docker image pushed up to the repo.
// If testing locally, user should run `make docker-build-debug` and interchaintest will use the local image.
func GetDockerImageInfo() (repo, version string) {
	branchVersion, found := os.LookupEnv("BRANCH_CI")
	if !found {
		// make local-image
		fmt.Println("Testing local image")
		repo = "orai"
		branchVersion = "debug"
	}

	// github converts / to - for pushed docker images
	branchVersion = strings.ReplaceAll(branchVersion, "/", "-")
	return repo, branchVersion
}

func modifyGenesisShortProposals(
	votingPeriod string,
	maxDepositPeriod string,
) func(ibc.ChainConfig, []byte) ([]byte, error) {
	return func(chainConfig ibc.ChainConfig, genbz []byte) ([]byte, error) {
		g := make(map[string]interface{})
		if err := json.Unmarshal(genbz, &g); err != nil {
			return nil, fmt.Errorf("failed to unmarshal genesis file: %w", err)
		}
		if err := dyno.Set(g, votingPeriod, "app_state", "gov", "params", "voting_period"); err != nil {
			return nil, fmt.Errorf("failed to set voting period in genesis json: %w", err)
		}
		if err := dyno.Set(g, "1", "initial_height"); err != nil {
			return nil, fmt.Errorf("failed to set initial height in genesis json: %w", err)
		}
		if err := dyno.Set(g, maxDepositPeriod, "app_state", "gov", "params", "max_deposit_period"); err != nil {
			return nil, fmt.Errorf("failed to set voting period in genesis json: %w", err)
		}
		if err := dyno.Set(g, chainConfig.Denom, "app_state", "gov", "params", "min_deposit", 0, "denom"); err != nil {
			return nil, fmt.Errorf("failed to set voting period in genesis json: %w", err)
		}
		out, err := json.Marshal(g)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal genesis bytes to json: %w", err)
		}
		return out, nil
	}
}
