package types

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"crypto/sha256"

	errorsmod "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	regexChainID         = `[a-z]{1,}`
	regexEIP155Separator = `_{1}`
	regexEIP155          = `[1-9][0-9]*`
	regexEpochSeparator  = `-{1}`
	regexEpoch           = `[1-9][0-9]*`
	ethermintChainID     = regexp.MustCompile(fmt.Sprintf(`^(%s)%s(%s)%s(%s)$`, regexChainID, regexEIP155Separator, regexEIP155, regexEpochSeparator, regexEpoch))
)

func hashChainIdToInt(chainID string) *big.Int {
	// Calculate the SHA256 hash of "Oraichain"
	hash := sha256.Sum256([]byte(chainID))

	// Convert the first 4 bytes to a big integer
	firstFourBytes := hash[:4]
	bigInt := new(big.Int).SetBytes(firstFourBytes)

	return bigInt
}

// IsValidChainID returns false if the given chain identifier is incorrectly formatted.
func IsValidChainID(chainID string) bool {
	chainID = strings.TrimSpace(chainID)
	if len(chainID) > 48 || len(chainID) == 0 {
		return false
	}

	// now we also support other types of chain ids by hashing and collecting the first 4 bytes
	// return ethermintChainID.MatchString(chainID)
	return true
}

// ParseChainID parses a string chain identifier's epoch to an Ethereum-compatible
// chain-id in *big.Int format. The function returns an error if the chain-id has an invalid format
func ParseChainID(chainID string) (*big.Int, error) {
	chainID = strings.TrimSpace(chainID)
	if len(chainID) > 48 {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidChainID, "chain-id '%s' cannot exceed 48 chars", chainID)
	}
	if len(chainID) == 0 {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidChainID, "chain-id '%s' cannot be empty", chainID)

	}
	matches := ethermintChainID.FindStringSubmatch(chainID)
	if matches == nil || len(matches) != 4 || matches[1] == "" {
		chainIDInt := hashChainIdToInt(chainID)
		return chainIDInt, nil
	}

	// verify that the chain-id entered is a base 10 integer
	chainIDInt, ok := new(big.Int).SetString(matches[2], 10)
	if !ok {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidChainID, "epoch %s must be base-10 integer format", matches[2])
	}

	return chainIDInt, nil
}
