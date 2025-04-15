package addr

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	"github.com/cosmos/evm/x/vm/core/vm"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func (p Precompile) Associate(
	ctx sdk.Context,
	contract *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	// v, r and s are components of a signature over the customMessage sent.
	// We use the signature to construct the user's pubkey to obtain their addresses.
	v, r, s, customMessage, err := ParseAssociateArgs(args)
	if err != nil {
		return nil, err
	}

	rBytes, err := decodeHexString(r)
	if err != nil {
		return nil, err
	}
	sBytes, err := decodeHexString(s)
	if err != nil {
		return nil, err
	}
	vBytes, err := decodeHexString(v)
	if err != nil {
		return nil, err
	}

	vBig := new(big.Int).SetBytes(vBytes)
	rBig := new(big.Int).SetBytes(rBytes)
	sBig := new(big.Int).SetBytes(sBytes)

	// Derive addresses
	vBig = new(big.Int).Add(vBig, big.NewInt(27))

	customMessageHash := crypto.Keccak256Hash([]byte(customMessage))
	pubKeyBytes, err := RecoverPubkey(customMessageHash, rBig, sBig, vBig, true)
	if err != nil {
		return nil, err
	}

	caller := contract.CallerAddress

	cosmosAddress, evmAddress, err := p.associateAddresses(ctx, caller, pubKeyBytes)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(cosmosAddress.String(), evmAddress)
}

func (p Precompile) AssociatePubKey(
	ctx sdk.Context,
	contract *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	pubKeyHex, err := ParseAssociatePubKeyArgs(args)
	if err != nil {
		return nil, err
	}

	pubKeyBytes, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return nil, err
	}

	caller := contract.CallerAddress

	cosmosAddress, evmAddress, err := p.associateAddresses(ctx, caller, pubKeyBytes)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(cosmosAddress.String(), evmAddress)
}

func (p Precompile) associateAddresses(
	ctx sdk.Context,
	caller common.Address,
	pubkey []byte,
) (sdk.AccAddress, *common.Address, error) {
	evmAddress, err := evmtypes.PubkeyBytesToEVMAddress(pubkey)
	if err != nil {
		return nil, nil, err
	}

	if evmAddress.Hex() != caller.Hex() {
		return nil, nil, fmt.Errorf("Caller address %s does not match with EVM address %s computed from the public key %s\n", caller.Hex(), evmAddress.Hex(), base64.StdEncoding.EncodeToString(pubkey))
	}

	cosmosAddress, err := evmtypes.PubkeyBytesToCosmosAddress(pubkey)
	if err != nil {
		return nil, nil, err
	}
	err = p.EVMKeeper.SetMappingEvmAddressInner(ctx, cosmosAddress.String(), base64.StdEncoding.EncodeToString(pubkey))
	return cosmosAddress, evmAddress, err
}
