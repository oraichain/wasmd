package addr

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	cmn "github.com/cosmos/evm/precompiles/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

func ParseAssociateArgs(
	args []interface{},
) (v string, r string, s string, customMessage string, err error) {
	if len(args) != 4 {
		return "", "", "", "", fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 4, len(args))
	}

	v, ok := args[0].(string)
	if !ok {
		return "", "", "", "", fmt.Errorf("invalid v: %v", args[0])
	}

	r, ok = args[1].(string)
	if !ok {
		return "", "", "", "", fmt.Errorf("invalid r: %v", args[1])
	}

	s, ok = args[2].(string)
	if !ok {
		return "", "", "", "", fmt.Errorf("invalid s: %v", args[2])
	}

	customMessage, ok = args[3].(string)
	if !ok {
		return "", "", "", "", fmt.Errorf("invalid customMessage: %v", args[3])
	}

	return v, r, s, customMessage, nil
}

func ParseAssociatePubKeyArgs(args []interface{}) (pubKeyHex string, err error) {
	if len(args) != 1 {
		return "", fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 1, len(args))
	}

	pubKeyHex, ok := args[0].(string)
	if !ok {
		return "", fmt.Errorf("invalid pubKeyHex: %v", args[0])
	}

	return pubKeyHex, nil
}

func ParseGetCosmosAddrArgs(args []interface{}) (evmAddress common.Address, err error) {
	if len(args) != 1 {
		return common.Address{}, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 1, len(args))
	}

	evmAddress, ok := args[0].(common.Address)
	if !ok {
		return common.Address{}, fmt.Errorf("invalid evmAddress: %v", args[0])
	}

	return evmAddress, nil
}

func ParseGetEvmAddrArgs(args []interface{}) (cosmosAddress string, err error) {
	if len(args) != 1 {
		return "", fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 1, len(args))
	}

	cosmosAddress, ok := args[0].(string)
	if !ok {
		return "", fmt.Errorf("invalid cosmosAddress: %v", args[0])
	}

	return cosmosAddress, nil
}

func decodeHexString(hexString string) ([]byte, error) {
	trimmed := strings.TrimPrefix(hexString, "0x")
	if len(trimmed)%2 != 0 {
		trimmed = "0" + trimmed
	}
	return hex.DecodeString(trimmed)
}

// first half of go-ethereum/core/types/transaction_signing.go:recoverPlain
func RecoverPubkey(sighash common.Hash, R, S, Vb *big.Int, homestead bool) ([]byte, error) {
	if Vb.BitLen() > 8 {
		return []byte{}, ethtypes.ErrInvalidSig
	}
	V := byte(Vb.Uint64() - 27)
	if !crypto.ValidateSignatureValues(V, R, S, homestead) {
		return []byte{}, ethtypes.ErrInvalidSig
	}
	// encode the signature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, crypto.SignatureLength)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V

	// recover the public key from the signature
	pubKeyBytes, err := crypto.Ecrecover(sighash[:], sig)
	if err != nil {
		return nil, err
	}
	btcecPubKey, err := btcec.ParsePubKey(pubKeyBytes)
	if err != nil {
		return nil, err
	}
	return btcecPubKey.SerializeCompressed(), nil
}
