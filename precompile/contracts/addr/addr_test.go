package addr_test

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"

	// "fmt"
	"math/big"
	"testing"

	"github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/precompile/contracts/addr"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/evm/x/vm/core/vm"
	"github.com/cosmos/evm/x/vm/statedb"
	"github.com/cosmos/go-bip39"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const suppliedGas = uint64(10_000_000)

func MockAddressPair() (sdk.AccAddress, common.Address) {
	return PrivateKeyToAddresses(MockPrivateKey())
}

func MockPrivateKey() cryptotypes.PrivKey {
	// Generate a new Sei private key
	entropySeed, _ := bip39.NewEntropy(256)
	mnemonic, _ := bip39.NewMnemonic(entropySeed)
	algo := hd.Secp256k1
	derivedPriv, _ := algo.Derive()(mnemonic, "", "")
	return algo.Generate()(derivedPriv)
}

func PrivateKeyToAddresses(privKey cryptotypes.PrivKey) (sdk.AccAddress, common.Address) {
	// Encode the private key to hex (i.e. what wallets do behind the scene when users reveal private keys)
	testPrivHex := hex.EncodeToString(privKey.Bytes())

	// Sign an Ethereum transaction with the hex private key
	key, _ := crypto.HexToECDSA(testPrivHex)
	msg := crypto.Keccak256([]byte("foo"))
	sig, _ := crypto.Sign(msg, key)

	// Recover the public keys from the Ethereum signature
	recoveredPub, _ := crypto.Ecrecover(msg, sig)
	pubKey, _ := crypto.UnmarshalPubkey(recoveredPub)

	return sdk.AccAddress(privKey.PubKey().Address()), crypto.PubkeyToAddress(*pubKey)
}

func TestGetCosmosAddr(t *testing.T) {
	tApp := app.Setup(t)
	ctx := tApp.NewContext(true)

	targetPrivKey := MockPrivateKey()
	targetCosmosAddress, targetEvmAddress := PrivateKeyToAddresses(targetPrivKey)
	targetCosmosAddressNoMapping := sdk.AccAddress(targetEvmAddress.Bytes())

	// set evm params
	EVMParams := tApp.GetEVMKeeper().GetParams(ctx)
	EVMParams.ActiveStaticPrecompiles = append(EVMParams.ActiveStaticPrecompiles, addr.AddrContractAddress)
	tApp.GetEVMKeeper().SetParams(ctx, EVMParams)

	p, found, err := tApp.GetEVMKeeper().GetPrecompileInstance(ctx, common.HexToAddress(addr.AddrContractAddress))
	require.True(t, found)
	require.NoError(t, err)

	contract := p.Map[common.HexToAddress(addr.AddrContractAddress)]
	require.NotNil(t, contract)

	suppliedGas := uint64(20_000_000)
	getCosmosAddrMethod := addr.ABI.Methods[addr.GetCosmosAddressMethod]

	happyPathOutputNoMapping, _ := getCosmosAddrMethod.Outputs.Pack(targetCosmosAddressNoMapping.String())
	happyPathOutput, _ := getCosmosAddrMethod.Outputs.Pack(targetCosmosAddress.String())

	type args struct {
		caller   common.Address
		value    *big.Int
		readOnly bool
		hookFn   func()
	}
	tests := []struct {
		name       string
		args       args
		wantRet    []byte
		wantErr    bool
		wantErrMsg string
		wrongRet   bool
	}{
		{
			name: "happy path - no evm mapping",
			args: args{
				caller: targetEvmAddress,
				value:  big.NewInt(0),
				hookFn: func() {},
			},
			wantRet: happyPathOutputNoMapping,
			wantErr: false,
		},
		{
			name: "happy path - with evm mapping",
			args: args{
				caller: targetEvmAddress,
				value:  big.NewInt(0),
				hookFn: func() {
					tApp.EvmKeeper.SetAddressMapping(ctx, targetCosmosAddress, targetEvmAddress)
				},
			},
			wantRet: happyPathOutput,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the EVM
			evm := vm.EVM{
				StateDB:   statedb.New(ctx, tApp.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))),
				TxContext: vm.TxContext{Origin: targetEvmAddress},
			}

			inputs, err := getCosmosAddrMethod.Inputs.Pack(tt.args.caller)
			require.Nil(t, err)

			// call hook before testing
			tt.args.hookFn()

			// Make the call to associate.
			ret, _, err := evm.RunPrecompiledContract(
				contract,
				vm.AccountRef(tt.args.caller),
				append(getCosmosAddrMethod.ID, inputs...),
				suppliedGas,
				nil,
				false,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v %v", err, tt.wantErr, string(ret))
				return
			}
			if err != nil {
				require.Equal(t, tt.wantErrMsg, err.Error())
			} else if tt.wrongRet {
				// tt.wrongRet is set if we expect a return value that's different from the happy path. This means that the wrong addresses were associated.
				require.NotEqual(t, tt.wantRet, ret)
			} else {
				require.Equal(t, tt.wantRet, ret)
			}
		})
	}
}

func TestGetEvmAddr(t *testing.T) {
	tApp := app.Setup(t)
	ctx := tApp.NewContext(true)

	targetPrivKey := MockPrivateKey()
	targetCosmosAddress, targetEvmAddress := PrivateKeyToAddresses(targetPrivKey)

	// set evm params
	EVMParams := tApp.GetEVMKeeper().GetParams(ctx)
	EVMParams.ActiveStaticPrecompiles = append(EVMParams.ActiveStaticPrecompiles, addr.AddrContractAddress)
	tApp.GetEVMKeeper().SetParams(ctx, EVMParams)

	p, found, err := tApp.GetEVMKeeper().GetPrecompileInstance(ctx, common.HexToAddress(addr.AddrContractAddress))
	require.True(t, found)
	require.NoError(t, err)

	contract := p.Map[common.HexToAddress(addr.AddrContractAddress)]
	require.NotNil(t, contract)

	suppliedGas := uint64(20_000_000)
	getEvmAddrMethod := addr.ABI.Methods[addr.GetEvmAddressMethod]

	happyPathOutput, _ := getEvmAddrMethod.Outputs.Pack(targetEvmAddress)

	type args struct {
		caller   common.Address
		value    *big.Int
		readOnly bool
		hookFn   func()
	}
	tests := []struct {
		name       string
		args       args
		wantRet    []byte
		wantErr    bool
		wantErrMsg string
		wrongRet   bool
	}{
		{
			name: "happy path - no evm mapping",
			args: args{
				caller: targetEvmAddress,
				value:  big.NewInt(0),
				hookFn: func() {},
			},
			wantErrMsg: fmt.Errorf("There is no evm address mapped to %s.", targetCosmosAddress).Error(),
			wantErr:    true,
		},
		{
			name: "happy path - with evm mapping",
			args: args{
				caller: targetEvmAddress,
				value:  big.NewInt(0),
				hookFn: func() {
					tApp.EvmKeeper.SetAddressMapping(ctx, targetCosmosAddress, targetEvmAddress)
				},
			},
			wantRet: happyPathOutput,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the EVM
			evm := vm.EVM{
				StateDB:   statedb.New(ctx, tApp.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))),
				TxContext: vm.TxContext{Origin: targetEvmAddress},
			}

			inputs, err := getEvmAddrMethod.Inputs.Pack(targetCosmosAddress.String())
			require.Nil(t, err)

			// call hook before testing
			tt.args.hookFn()

			// Make the call to associate.
			ret, _, err := evm.RunPrecompiledContract(
				contract,
				vm.AccountRef(tt.args.caller),
				append(getEvmAddrMethod.ID, inputs...),
				suppliedGas,
				nil,
				false,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v %v", err, tt.wantErr, string(ret))
				return
			}
			if err != nil {
				require.Equal(t, tt.wantErrMsg, err.Error())
			} else if tt.wrongRet {
				// tt.wrongRet is set if we expect a return value that's different from the happy path. This means that the wrong addresses were associated.
				require.NotEqual(t, tt.wantRet, ret)
			} else {
				require.Equal(t, tt.wantRet, ret)
			}
		})
	}
}

func TestAssociatePubKey(t *testing.T) {
	tApp := app.Setup(t)
	ctx := tApp.NewContext(true)

	// Target refers to the address that the caller is trying to associate.
	targetPrivKey := MockPrivateKey()
	targetPubKey := targetPrivKey.PubKey()
	targetPubKeyHex := hex.EncodeToString(targetPubKey.Bytes())
	targetCosmosAddress, targetEvmAddress := PrivateKeyToAddresses(targetPrivKey)

	// Caller refers to the party calling the precompile.
	callerPrivKey := MockPrivateKey()
	_, callerEvmAddress := PrivateKeyToAddresses(callerPrivKey)

	// set evm params
	EVMParams := tApp.GetEVMKeeper().GetParams(ctx)
	EVMParams.ActiveStaticPrecompiles = append(EVMParams.ActiveStaticPrecompiles, addr.AddrContractAddress)
	tApp.GetEVMKeeper().SetParams(ctx, EVMParams)

	p, found, err := tApp.GetEVMKeeper().GetPrecompileInstance(ctx, common.HexToAddress(addr.AddrContractAddress))
	require.True(t, found)
	require.NoError(t, err)

	contract := p.Map[common.HexToAddress(addr.AddrContractAddress)]
	require.NotNil(t, contract)

	suppliedGas := uint64(20_000_000)
	associatePubKeyMethod := addr.ABI.Methods[addr.AssociatePubKeyMethod]

	happyPathOutput, _ := associatePubKeyMethod.Outputs.Pack(targetCosmosAddress.String(), targetEvmAddress)

	type args struct {
		caller common.Address
		pubKey string
		value  *big.Int
	}
	tests := []struct {
		name       string
		args       args
		wantRet    []byte
		wantErr    bool
		wantErrMsg string
		wrongRet   bool
	}{
		// {
		// 	name: "fails if payable",
		// 	args: args{
		// 		caller: callerEvmAddress,
		// 		pubKey: hex.EncodeToString(callerPrivKey.PubKey().Bytes()),
		// 		value:  big.NewInt(10),
		// 	},
		// 	wantErr:    true,
		// 	wantErrMsg: "sending funds to a non-payable function",
		// },
		// {
		// 	name: "fails on static call",
		// 	args: args{
		// 		caller:   callerEvmAddress,
		// 		pubKey:   hex.EncodeToString(callerPrivKey.PubKey().Bytes()),
		// 		value:    big.NewInt(10),
		// 		readOnly: true,
		// 	},
		// 	wantErr:    true,
		// 	wantErrMsg: "cannot call associate pub key precompile from staticcall",
		// },
		{
			name: "fails if input is appended with 0x",
			args: args{
				caller: callerEvmAddress,
				pubKey: fmt.Sprintf("0x%v", targetPubKeyHex),
				value:  big.NewInt(0),
			},
			wantErr:    true,
			wantErrMsg: "encoding/hex: invalid byte: U+0078 'x'",
		},
		{
			name: "fails if caller address does not match with public key",
			args: args{
				caller: callerEvmAddress,
				pubKey: targetPubKeyHex,
				value:  big.NewInt(0),
			},
			wantErrMsg: fmt.Errorf("Caller address %s does not match with EVM address %s computed from the public key %s\n", callerEvmAddress.Hex(), targetEvmAddress.Hex(), base64.StdEncoding.EncodeToString(targetPubKey.Bytes())).Error(),
			wantErr:    true,
		},
		{
			name: "happy path - associates addresses if signature is correct",
			args: args{
				caller: targetEvmAddress,
				pubKey: targetPubKeyHex,
				value:  big.NewInt(0),
			},
			wantRet: happyPathOutput,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the EVM
			evm := vm.EVM{
				StateDB:   statedb.New(ctx, tApp.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))),
				TxContext: vm.TxContext{Origin: callerEvmAddress},
			}

			// Create the precompile and inputs
			inputs, err := associatePubKeyMethod.Inputs.Pack(tt.args.pubKey)
			require.Nil(t, err)

			// Make the call to associate.
			ret, _, err := evm.RunPrecompiledContract(
				contract,
				vm.AccountRef(tt.args.caller),
				append(associatePubKeyMethod.ID, inputs...),
				suppliedGas,
				tt.args.value,
				false,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v %v", err, tt.wantErr, string(ret))
				return
			}
			if err != nil {
				require.Equal(t, tt.wantErrMsg, err.Error())
			} else if tt.wrongRet {
				// tt.wrongRet is set if we expect a return value that's different from the happy path. This means that the wrong addresses were associated.
				require.NotEqual(t, tt.wantRet, ret)
			} else {
				// Because evm using cache context, we need to get the cache context
				cacheCtx, err := evm.StateDB.(*statedb.StateDB).GetCacheContext()
				require.NoError(t, err)

				require.Equal(t, tt.wantRet, ret)
				mappedCosmosAddress := tApp.EvmKeeper.GetCosmosAddressMapping(cacheCtx, targetEvmAddress)
				require.Equal(t, targetCosmosAddress, mappedCosmosAddress)
				mappedEvmAddress, err := tApp.EvmKeeper.GetEvmAddressMapping(cacheCtx, targetCosmosAddress)
				require.NoError(t, err)
				require.Equal(t, &targetEvmAddress, mappedEvmAddress)
			}
		})
	}
}

func TestAssociate(t *testing.T) {
	tApp := app.Setup(t)
	ctx := tApp.NewContext(true)

	// Target refers to the address that the caller is trying to associate.
	targetPrivKey := MockPrivateKey()
	targetPrivHex := hex.EncodeToString(targetPrivKey.Bytes())
	targetCosmosAddress, targetEvmAddress := PrivateKeyToAddresses(targetPrivKey)
	targetKey, _ := crypto.HexToECDSA(targetPrivHex)

	// Create the inputs
	emptyData := make([]byte, 32)
	prefixedMessage := fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(emptyData)) + string(emptyData)
	hash := crypto.Keccak256Hash([]byte(prefixedMessage))
	sig, err := crypto.Sign(hash.Bytes(), targetKey)
	require.Nil(t, err)

	r := fmt.Sprintf("0x%v", new(big.Int).SetBytes(sig[:32]).Text(16))
	s := fmt.Sprintf("0x%v", new(big.Int).SetBytes(sig[32:64]).Text(16))
	v := fmt.Sprintf("0x%v", new(big.Int).SetBytes([]byte{sig[64]}).Text(16))

	// Caller refers to the party calling the precompile.
	callerPrivKey := MockPrivateKey()
	_, callerEvmAddress := PrivateKeyToAddresses(callerPrivKey)

	// set evm params
	EVMParams := tApp.GetEVMKeeper().GetParams(ctx)
	EVMParams.ActiveStaticPrecompiles = append(EVMParams.ActiveStaticPrecompiles, addr.AddrContractAddress)
	tApp.GetEVMKeeper().SetParams(ctx, EVMParams)

	p, found, err := tApp.GetEVMKeeper().GetPrecompileInstance(ctx, common.HexToAddress(addr.AddrContractAddress))
	require.True(t, found)
	require.NoError(t, err)

	contract := p.Map[common.HexToAddress(addr.AddrContractAddress)]
	require.NotNil(t, contract)

	suppliedGas := uint64(20_000_000)
	associateMethod := addr.ABI.Methods[addr.AssociateMethod]

	happyPathOutput, _ := associateMethod.Outputs.Pack(targetCosmosAddress.String(), targetEvmAddress)

	type args struct {
		caller common.Address
		v      string
		r      string
		s      string
		msg    string
		value  *big.Int
	}
	tests := []struct {
		name       string
		args       args
		wantRet    []byte
		wantErr    bool
		wantErrMsg string
		wrongRet   bool
	}{
		// {
		// 	name: "fails if payable",
		// 	args: args{
		// 		caller: callerEvmAddress,
		// 		v:      v,
		// 		r:      r,
		// 		s:      s,
		// 		msg:    prefixedMessage,
		// 		value:  big.NewInt(10),
		// 	},
		// 	wantErr:    true,
		// 	wantErrMsg: "sending funds to a non-payable function",
		// },
		// {
		// 	name: "fails on static calls",
		// 	args: args{
		// 		caller: callerEvmAddress,
		// 		v:      v,
		// 		r:      r,
		// 		s:      s,
		// 		msg:    prefixedMessage,
		// 		value:  big.NewInt(10),
		// 	},
		// 	wantErr:    true,
		// 	wantErrMsg: "cannot call associate precompile from staticcall",
		// },
		{
			name: "fails if input is not hex",
			args: args{
				caller: callerEvmAddress,
				v:      "nothex",
				r:      r,
				s:      s,
				msg:    prefixedMessage,
				value:  big.NewInt(0),
			},
			wantErr:    true,
			wantErrMsg: "encoding/hex: invalid byte: U+006E 'n'",
		},
		{
			name: "associates wrong address if invalid signature (different message)",
			args: args{
				caller: callerEvmAddress,
				v:      v,
				r:      r,
				s:      s, // Pass in r instead of s here for invalid value
				msg:    prefixedMessage,
				value:  big.NewInt(0),
			},
			wantErrMsg: fmt.Errorf("Caller address %s does not match with EVM address %s computed from the public key %s\n", callerEvmAddress.Hex(), targetEvmAddress.Hex(), base64.StdEncoding.EncodeToString(targetPrivKey.PubKey().Bytes())).Error(),
			wantErr:    true,
		},
		{
			name: "happy path - associates addresses if signature is correct",
			args: args{
				caller: targetEvmAddress,
				v:      v,
				r:      r,
				s:      s,
				msg:    prefixedMessage,
				value:  big.NewInt(0),
			},
			wantRet: happyPathOutput,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the EVM
			evm := vm.EVM{
				StateDB:   statedb.New(ctx, tApp.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))),
				TxContext: vm.TxContext{Origin: callerEvmAddress},
			}

			// Create the precompile and inputs
			inputs, err := associateMethod.Inputs.Pack(tt.args.v, tt.args.r, tt.args.s, tt.args.msg)
			require.Nil(t, err)

			// Make the call to associate.
			ret, _, err := evm.RunPrecompiledContract(
				contract,
				vm.AccountRef(tt.args.caller),
				append(associateMethod.ID, inputs...),
				suppliedGas,
				tt.args.value,
				false,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v %v", err, tt.wantErr, string(ret))
				return
			}
			if err != nil {
				require.Equal(t, tt.wantErrMsg, err.Error())
			} else if tt.wrongRet {
				// tt.wrongRet is set if we expect a return value that's different from the happy path. This means that the wrong addresses were associated.
				require.NotEqual(t, tt.wantRet, ret)
			} else {
				// Because evm using cache context, we need to get the cache context
				cacheCtx, err := evm.StateDB.(*statedb.StateDB).GetCacheContext()
				require.NoError(t, err)

				require.Equal(t, tt.wantRet, ret)
				mappedCosmosAddress := tApp.EvmKeeper.GetCosmosAddressMapping(cacheCtx, targetEvmAddress)
				require.Equal(t, targetCosmosAddress, mappedCosmosAddress)
				mappedEvmAddress, err := tApp.EvmKeeper.GetEvmAddressMapping(cacheCtx, targetCosmosAddress)
				require.NoError(t, err)
				require.Equal(t, &targetEvmAddress, mappedEvmAddress)
			}
		})
	}
}
