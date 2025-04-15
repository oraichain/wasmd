package addr

import (
	_ "embed"
	"fmt"

	pcommon "github.com/CosmWasm/wasmd/precompile/common"
	cmn "github.com/cosmos/evm/precompiles/common"

	"github.com/cosmos/evm/x/vm/core/vm"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/precompile/contract"
)

// Singleton StatefulPrecompiledContract.
var (
	// RawABI contains the raw ABI of addr contract.
	//go:embed abi.json
	RawABI string

	ABI = contract.MustParseABI(RawABI)
)

const (
	AddrContractAddress = "0x9000000000000000000000000000000000000003"

	// Define the minumum gas required needed for each method
	// TODO: need to re-define gas required here
	AssociateMethodRequiredGas        = 5_000_000
	AssociatePubKeyMethodRequiredGas  = 5_000_000
	GetCosmosAddressMethodRequiredGas = 30_000
	GetEvmAddressMethodRequiredGas    = 30_000

	// Execute methods
	AssociateMethod       = "associate"
	AssociatePubKeyMethod = "associatePubKey"

	// Query methods
	GetCosmosAddressMethod = "getCosmosAddr"
	GetEvmAddressMethod    = "getEvmAddr"
)

// Precompile defines the precompiled contract for addr.
type Precompile struct {
	cmn.Precompile
	EVMKeeper pcommon.EVMKeeper
}

func NewPrecompile(evmKeeper pcommon.EVMKeeper) (*Precompile, error) {
	p := &Precompile{
		Precompile: cmn.Precompile{
			ABI: ABI,
		},
		EVMKeeper: evmKeeper,
	}

	// SetAddress defines the address of the addr compile contract.
	p.SetAddress(common.HexToAddress(AddrContractAddress))

	return p, nil
}

func (p Precompile) Address() common.Address {
	return p.Precompile.Address()
}

// RequiredGas calculates the precompiled contract's base gas rate.
func (p Precompile) RequiredGas(input []byte) uint64 {
	if len(input) < 4 {
		return 0
	}
	methodID := input[:4]

	method, err := p.MethodById(methodID)
	if err != nil {
		return 0
	}

	switch method.Name {
	case AssociateMethod:
		return AssociateMethodRequiredGas
	case AssociatePubKeyMethod:
		return AssociatePubKeyMethodRequiredGas
	case GetCosmosAddressMethod:
		return GetCosmosAddressMethodRequiredGas
	case GetEvmAddressMethod:
		return GetEvmAddressMethodRequiredGas
	}

	return 0
}

// Run executes the precompiled contract addr methods defined in the ABI.
func (p Precompile) Run(
	evm *vm.EVM,
	contract *vm.Contract,
	readOnly bool,
) (bz []byte, err error) {
	ctx, stateDB, snapshot, method, initialGas, args, err := p.RunSetup(evm, contract, readOnly, p.IsTransaction)
	if err != nil {
		return nil, err
	}

	// This handles any out of gas errors that may occur during the execution of a precompile.
	// It avoids panics and returns the out of gas error so the EVM can continue gracefully.
	defer cmn.HandleGasError(ctx, contract, initialGas, &err)()

	switch method.Name {
	case AssociateMethod:
		bz, err = p.Associate(ctx, contract, method, args)
		break
	case AssociatePubKeyMethod:
		bz, err = p.AssociatePubKey(ctx, contract, method, args)
		break
	case GetCosmosAddressMethod:
		bz, err = p.GetCosmosAddr(ctx, contract, method, args)
		break
	case GetEvmAddressMethod:
		bz, err = p.GetEvmAddr(ctx, contract, method, args)
		break
	default:
		return nil, fmt.Errorf(cmn.ErrUnknownMethod, method.Name)
	}

	if err != nil {
		return nil, err
	}

	cost := ctx.GasMeter().GasConsumed() - initialGas

	if !contract.UseGas(cost) {
		return nil, vm.ErrOutOfGas
	}

	if err := p.AddJournalEntries(stateDB, snapshot); err != nil {
		return nil, err
	}

	return bz, nil
}

// IsTransaction checks if the given method name corresponds to a transaction or query.
func (Precompile) IsTransaction(method *abi.Method) bool {
	switch method.Name {
	case AssociateMethod:
	case AssociatePubKeyMethod:
		return true
	default:
		return false
	}

	return false
}

// func (p PrecompileExecutor) getCosmosAddr(accessibleState contract.AccessibleState,
// 	caller common.Address,
// 	callingContract common.Address,
// 	packedInput []byte,
// 	suppliedGas uint64,
// 	readOnly bool,
// 	value *big.Int) (ret []byte, remainingGas uint64, rerr error) {

// 	ctx, initialGas, rerr := pcommon.GetPrecompileCtx(accessibleState)
// 	if rerr != nil {
// 		return
// 	}

// 	defer func() {
// 		if err := recover(); err != nil {
// 			ret = nil
// 			remainingGas = 0
// 			rerr = fmt.Errorf("%s", err)
// 			ctx.Logger().Error("Error querying getCosmosAddr using precompile: ", rerr.Error())
// 			return
// 		}
// 	}()
// 	method := ABI.Methods[GetCosmosAddressMethod]

// 	args, err := method.Inputs.Unpack(packedInput)
// 	if err != nil {
// 		rerr = err
// 		return
// 	}

// 	if err := pcommon.ValidateNonPayable(value); err != nil {
// 		rerr = err
// 		return
// 	}

// 	if err := pcommon.ValidateArgsLength(args, 1); err != nil {
// 		rerr = err
// 		return
// 	}
// 	evmAddress := args[0].(common.Address)

// 	cosmosAddress := p.evmKeeper.GetCosmosAddressMapping(ctx, evmAddress)

// 	ret, rerr = method.Outputs.Pack(cosmosAddress.String())
// 	remainingGas, rerr = contract.DeductGas(suppliedGas, ctx.GasMeter().GasConsumed()-initialGas)
// 	return
// }

// func (p PrecompileExecutor) getEvmAddr(accessibleState contract.AccessibleState,
// 	caller common.Address,
// 	callingContract common.Address,
// 	packedInput []byte,
// 	suppliedGas uint64,
// 	readOnly bool,
// 	value *big.Int) (ret []byte, remainingGas uint64, rerr error) {

// 	ctx, initialGas, rerr := pcommon.GetPrecompileCtx(accessibleState)
// 	if rerr != nil {
// 		return
// 	}

// 	defer func() {
// 		if err := recover(); err != nil {
// 			ret = nil
// 			remainingGas = 0
// 			rerr = fmt.Errorf("%s\n", err)
// 			ctx.Logger().Error("Error querying getEvmAddr using precompile: ", rerr.Error())
// 			return
// 		}
// 	}()
// 	method := ABI.Methods[GetEvmAddressMethod]

// 	args, err := method.Inputs.Unpack(packedInput)
// 	if err != nil {
// 		rerr = err
// 		return
// 	}

// 	if err := pcommon.ValidateNonPayable(value); err != nil {
// 		rerr = err
// 		return
// 	}

// 	if err := pcommon.ValidateArgsLength(args, 1); err != nil {
// 		rerr = err
// 		return
// 	}

// 	cosmosAddress, err := sdk.AccAddressFromBech32(args[0].(string))
// 	if err != nil {
// 		rerr = err
// 		return
// 	}

// 	evmAddress, err := p.evmKeeper.GetEvmAddressMapping(ctx, cosmosAddress)
// 	if err != nil {
// 		rerr = fmt.Errorf("cosmos address %s is not associated\n", cosmosAddress)
// 		return
// 	}

// 	ret, rerr = method.Outputs.Pack(evmAddress)
// 	remainingGas, rerr = contract.DeductGas(suppliedGas, ctx.GasMeter().GasConsumed()-initialGas)
// 	return
// }

// func (p PrecompileExecutor) associate(accessibleState contract.AccessibleState,
// 	caller common.Address,
// 	callingContract common.Address,
// 	packedInput []byte,
// 	suppliedGas uint64,
// 	readOnly bool,
// 	value *big.Int) (ret []byte, remainingGas uint64, rerr error) {

// 	ctx, initialGas, rerr := pcommon.GetPrecompileCtx(accessibleState)
// 	if rerr != nil {
// 		return
// 	}

// 	defer func() {
// 		if err := recover(); err != nil {
// 			ret = nil
// 			remainingGas = 0
// 			rerr = fmt.Errorf("%s\n", err)
// 			ctx.Logger().Error("Error associating using precompile: ", rerr.Error())
// 			return
// 		}
// 	}()

// 	if readOnly {
// 		rerr = errors.New("cannot call associate precompile from staticcall")
// 		return
// 	}

// 	method := ABI.Methods[AssociateMethod]

// 	args, err := method.Inputs.Unpack(packedInput)
// 	if err != nil {
// 		rerr = err
// 		return
// 	}

// 	if err := pcommon.ValidateNonPayable(value); err != nil {
// 		rerr = err
// 		return
// 	}

// 	if err := pcommon.ValidateArgsLength(args, 4); err != nil {
// 		rerr = err
// 		return
// 	}

// 	// v, r and s are components of a signature over the customMessage sent.
// 	// We use the signature to construct the user's pubkey to obtain their addresses.
// 	v := args[0].(string)
// 	r := args[1].(string)
// 	s := args[2].(string)
// 	customMessage := args[3].(string)

// 	rBytes, err := decodeHexString(r)
// 	if err != nil {
// 		rerr = err
// 		return
// 	}
// 	sBytes, err := decodeHexString(s)
// 	if err != nil {
// 		rerr = err
// 		return
// 	}
// 	vBytes, err := decodeHexString(v)
// 	if err != nil {
// 		rerr = err
// 		return
// 	}

// 	vBig := new(big.Int).SetBytes(vBytes)
// 	rBig := new(big.Int).SetBytes(rBytes)
// 	sBig := new(big.Int).SetBytes(sBytes)

// 	// Derive addresses
// 	vBig = new(big.Int).Add(vBig, big.NewInt(27))

// 	customMessageHash := crypto.Keccak256Hash([]byte(customMessage))
// 	pubKeyBytes, err := RecoverPubkey(customMessageHash, rBig, sBig, vBig, true)
// 	if err != nil {
// 		rerr = err
// 		return
// 	}

// 	cosmosAddress, evmAddress, err := p.associateAddresses(ctx, caller, pubKeyBytes)
// 	if err != nil {
// 		rerr = err
// 		return
// 	}

// 	ret, rerr = method.Outputs.Pack(cosmosAddress.String(), evmAddress)
// 	remainingGas, rerr = contract.DeductGas(suppliedGas, ctx.GasMeter().GasConsumed()-initialGas)
// 	return
// }

// func (p PrecompileExecutor) associatePublicKey(accessibleState contract.AccessibleState,
// 	caller common.Address,
// 	callingContract common.Address,
// 	packedInput []byte,
// 	suppliedGas uint64,
// 	readOnly bool,
// 	value *big.Int) (ret []byte, remainingGas uint64, rerr error) {

// 	ctx, initialGas, rerr := pcommon.GetPrecompileCtx(accessibleState)
// 	if rerr != nil {
// 		return
// 	}

// 	defer func() {
// 		if err := recover(); err != nil {
// 			ret = nil
// 			remainingGas = 0
// 			rerr = fmt.Errorf("%s\n", err)
// 			ctx.Logger().Error("Error associating public key using precompile: ", rerr.Error())
// 			return
// 		}
// 	}()

// 	if readOnly {
// 		rerr = errors.New("cannot call associate pub key precompile from staticcall")
// 		return
// 	}

// 	method := ABI.Methods[AssociatePubKeyMethod]

// 	args, err := method.Inputs.Unpack(packedInput)
// 	if err != nil {
// 		rerr = err
// 		return
// 	}

// 	if err := pcommon.ValidateNonPayable(value); err != nil {
// 		rerr = err
// 		return
// 	}

// 	if err := pcommon.ValidateArgsLength(args, 1); err != nil {
// 		rerr = err
// 		return
// 	}

// 	// Takes a single argument, a compressed pubkey in hex format, excluding the '0x'
// 	pubKeyHex := args[0].(string)
// 	pubKeyBytes, err := hex.DecodeString(pubKeyHex)
// 	if err != nil {
// 		rerr = err
// 		return
// 	}

// 	cosmosAddress, evmAddress, err := p.associateAddresses(ctx, caller, pubKeyBytes)
// 	if err != nil {
// 		rerr = err
// 		return
// 	}

// 	ret, rerr = method.Outputs.Pack(cosmosAddress.String(), evmAddress)
// 	remainingGas, rerr = contract.DeductGas(suppliedGas, ctx.GasMeter().GasConsumed()-initialGas)
// 	return
// }

// func (p PrecompileExecutor) associateAddresses(ctx sdk.Context, caller common.Address, pubkey []byte) (sdk.AccAddress, *common.Address, error) {
// 	evmAddress, err := evmtypes.PubkeyBytesToEVMAddress(pubkey)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	if evmAddress.Hex() != caller.Hex() {
// 		return nil, nil, fmt.Errorf("Caller address %s does not match with EVM address %s computed from the public key %s\n", caller.Hex(), evmAddress.Hex(), base64.StdEncoding.EncodeToString(pubkey))
// 	}

// 	cosmosAddress, err := evmtypes.PubkeyBytesToCosmosAddress(pubkey)
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	err = p.evmKeeper.SetMappingEvmAddressInner(ctx, cosmosAddress.String(), base64.StdEncoding.EncodeToString(pubkey))
// 	return cosmosAddress, evmAddress, err
// }

// func decodeHexString(hexString string) ([]byte, error) {
// 	trimmed := strings.TrimPrefix(hexString, "0x")
// 	if len(trimmed)%2 != 0 {
// 		trimmed = "0" + trimmed
// 	}
// 	return hex.DecodeString(trimmed)
// }

// // first half of go-ethereum/core/types/transaction_signing.go:recoverPlain
// func RecoverPubkey(sighash common.Hash, R, S, Vb *big.Int, homestead bool) ([]byte, error) {
// 	if Vb.BitLen() > 8 {
// 		return []byte{}, ethtypes.ErrInvalidSig
// 	}
// 	V := byte(Vb.Uint64() - 27)
// 	if !crypto.ValidateSignatureValues(V, R, S, homestead) {
// 		return []byte{}, ethtypes.ErrInvalidSig
// 	}
// 	// encode the signature in uncompressed format
// 	r, s := R.Bytes(), S.Bytes()
// 	sig := make([]byte, crypto.SignatureLength)
// 	copy(sig[32-len(r):32], r)
// 	copy(sig[64-len(s):64], s)
// 	sig[64] = V

// 	// recover the public key from the signature
// 	pubKeyBytes, err := crypto.Ecrecover(sighash[:], sig)
// 	if err != nil {
// 		return nil, err
// 	}
// 	btcecPubKey, err := btcec.ParsePubKey(pubKeyBytes)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return btcecPubKey.SerializeCompressed(), nil
// }
