package authz_test

import (
	"encoding/hex"
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	"github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/precompile/contracts/authz"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/evm/x/vm/core/vm"
	"github.com/cosmos/evm/x/vm/statedb"
	"github.com/cosmos/go-bip39"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

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

func TestSetGrant(t *testing.T) {
	denom := "orai"
	tApp := app.Setup(t)
	ctx := tApp.NewContext(true)
	sdk.RegisterDenom(denom, sdkmath.LegacyNewDec(6))

	granterAddr, granterEvmAddr := MockAddressPair()
	granteeAddr, granteeEvmAddr := MockAddressPair()
	tApp.EvmKeeper.SetAddressMapping(ctx, granterAddr, granterEvmAddr)
	tApp.EvmKeeper.SetAddressMapping(ctx, granteeAddr, granteeEvmAddr)

	mintCoins := sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewInt(100000)))
	grantCoins := sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewInt(100)))
	bankKeeper := tApp.GetBankKeeper()
	authzKeeper := tApp.GetAuthzKeeper()
	err := bankKeeper.MintCoins(ctx, evmtypes.ModuleName, mintCoins)
	require.NoError(t, err)
	tApp.GetBankKeeper().SendCoinsFromModuleToAccount(ctx, evmtypes.ModuleName, granterAddr, grantCoins)
	tApp.GetBankKeeper().SendCoinsFromModuleToAccount(ctx, evmtypes.ModuleName, granteeAddr, grantCoins)
	tApp.GetBankKeeper().SetParams(ctx, banktypes.DefaultParams())

	// set evm params
	EVMParams := tApp.GetEVMKeeper().GetParams(ctx)
	EVMParams.ActiveStaticPrecompiles = append(EVMParams.ActiveStaticPrecompiles, authz.AuthzContractAddress)
	tApp.GetEVMKeeper().SetParams(ctx, EVMParams)

	p, found, err := tApp.GetEVMKeeper().GetPrecompileInstance(ctx, common.HexToAddress(authz.AuthzContractAddress))
	require.True(t, found)
	require.NoError(t, err)

	contract := p.Map[common.HexToAddress(authz.AuthzContractAddress)]
	require.NotNil(t, contract)

	suppliedGas := uint64(20_000_000)
	setGrantMethod := authz.ABI.Methods[authz.SetGrantMethod]

	// Create the EVM
	evm := vm.EVM{
		StateDB: statedb.New(ctx, tApp.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))),
	}

	args, err := setGrantMethod.Inputs.Pack(granteeEvmAddr, denom, grantCoins[0].Amount.BigInt())
	require.Nil(t, err)
	res, _, err := evm.RunPrecompiledContract(
		contract,
		vm.AccountRef(granterEvmAddr),
		append(setGrantMethod.ID, args...),
		suppliedGas,
		nil,
		false,
	)
	require.Nil(t, err)
	output, err := setGrantMethod.Outputs.Unpack(res)
	require.Nil(t, err)
	require.Equal(t, 1, len(output))
	require.Equal(t, output[0].(bool), true)

	grantMsg := &authztypes.QueryGrantsRequest{
		Granter:    granterAddr.String(),
		Grantee:    granteeAddr.String(),
		MsgTypeUrl: banktypes.SendAuthorization{}.MsgTypeURL(),
		Pagination: nil,
	}

	// Because evm using cache context, we need to get the cache context
	cacheCtx, err := evm.StateDB.(*statedb.StateDB).GetCacheContext()
	require.NoError(t, err)

	grant, err := authzKeeper.Grants(cacheCtx, grantMsg)
	require.Nil(t, err)
	require.Equal(t, 1, len(grant.Grants))

	var sendAuthorization banktypes.SendAuthorization
	var grantCoin sdk.Coin

	for _, g := range grant.Grants {
		sendAuthorization.Unmarshal(g.Authorization.Value)

		for _, coin := range sendAuthorization.SpendLimit {
			if coin.Denom == denom {
				grantCoin = coin
			}
		}
	}

	require.Equal(t, grantCoin.Denom, denom)
	require.Equal(t, grantCoin.Amount, grantCoins[0].Amount)
}

func TestQueryGrant(t *testing.T) {
	denom := "orai"
	tApp := app.Setup(t)
	ctx := tApp.NewContext(true)
	sdk.RegisterDenom(denom, sdkmath.LegacyNewDec(6))

	granterAddr, granterEvmAddr := MockAddressPair()
	granteeAddr, granteeEvmAddr := MockAddressPair()
	cosmosAddr, evmAddr := MockAddressPair()
	tApp.EvmKeeper.SetAddressMapping(ctx, granterAddr, granterEvmAddr)
	tApp.EvmKeeper.SetAddressMapping(ctx, granteeAddr, granteeEvmAddr)
	tApp.EvmKeeper.SetAddressMapping(ctx, cosmosAddr, evmAddr)

	mintCoins := sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewInt(100000)))
	grantCoins := sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewInt(100)))
	bankKeeper := tApp.GetBankKeeper()
	authzKeeper := tApp.GetAuthzKeeper()
	err := bankKeeper.MintCoins(ctx, evmtypes.ModuleName, mintCoins)
	require.NoError(t, err)
	tApp.GetBankKeeper().SendCoinsFromModuleToAccount(ctx, evmtypes.ModuleName, granterAddr, grantCoins)
	tApp.GetBankKeeper().SendCoinsFromModuleToAccount(ctx, evmtypes.ModuleName, granteeAddr, grantCoins)
	tApp.GetBankKeeper().SendCoinsFromModuleToAccount(ctx, evmtypes.ModuleName, cosmosAddr, grantCoins)
	tApp.GetBankKeeper().SetParams(ctx, banktypes.DefaultParams())

	// grant
	authorization := banktypes.NewSendAuthorization(grantCoins, []sdk.AccAddress{})
	setGrantMsg, err := authztypes.NewMsgGrant(granterAddr, granteeAddr, authorization, nil)
	require.NoError(t, err)

	_, err = authzKeeper.Grant(ctx, setGrantMsg)
	require.NoError(t, err)

	// set evm params
	EVMParams := tApp.GetEVMKeeper().GetParams(ctx)
	EVMParams.ActiveStaticPrecompiles = append(EVMParams.ActiveStaticPrecompiles, authz.AuthzContractAddress)
	tApp.GetEVMKeeper().SetParams(ctx, EVMParams)

	p, found, err := tApp.GetEVMKeeper().GetPrecompileInstance(ctx, common.HexToAddress(authz.AuthzContractAddress))
	require.True(t, found)
	require.NoError(t, err)

	contract := p.Map[common.HexToAddress(authz.AuthzContractAddress)]
	require.NotNil(t, contract)

	suppliedGas := uint64(20_000_000)
	grantMethod := authz.ABI.Methods[authz.GrantMethod]

	// Create the EVM
	evm := vm.EVM{
		StateDB: statedb.New(ctx, tApp.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))),
	}

	// have grant
	args, err := grantMethod.Inputs.Pack(granterEvmAddr, granteeEvmAddr, denom)
	require.Nil(t, err)
	res, _, err := evm.RunPrecompiledContract(
		contract,
		vm.AccountRef(granteeEvmAddr),
		append(grantMethod.ID, args...),
		suppliedGas,
		nil,
		false,
	)
	require.Nil(t, err)
	output, err := grantMethod.Outputs.Unpack(res)
	require.Nil(t, err)
	require.Equal(t, 1, len(output))
	require.Equal(t, output[0].(*big.Int), big.NewInt(grantCoins[0].Amount.Int64()))

	// no grant
	args, err = grantMethod.Inputs.Pack(granterEvmAddr, evmAddr, denom)
	require.Nil(t, err)
	res, _, err = evm.RunPrecompiledContract(
		contract,
		vm.AccountRef(evmAddr),
		append(grantMethod.ID, args...),
		suppliedGas,
		nil,
		false,
	)
	require.Nil(t, err)
	output, err = grantMethod.Outputs.Unpack(res)
	require.Nil(t, err)
	require.Equal(t, 1, len(output))
	require.Equal(t, output[0].(*big.Int).Int64(), big.NewInt(0).Int64())
}

func TestExecGrant(t *testing.T) {
	denom := "orai"
	tApp := app.Setup(t)
	ctx := tApp.NewContext(true)
	sdk.RegisterDenom(denom, sdkmath.LegacyNewDec(6))

	granterAddr, granterEvmAddr := MockAddressPair()
	granteeAddr, granteeEvmAddr := MockAddressPair()
	recipientAddr, recipientEvmAddr := MockAddressPair()
	tApp.EvmKeeper.SetAddressMapping(ctx, granterAddr, granterEvmAddr)
	tApp.EvmKeeper.SetAddressMapping(ctx, granteeAddr, granteeEvmAddr)
	tApp.EvmKeeper.SetAddressMapping(ctx, recipientAddr, recipientEvmAddr)

	mintCoins := sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewInt(100000)))
	grantCoins := sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewInt(100)))
	transferCoins := sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewInt(10)))
	bankKeeper := tApp.GetBankKeeper()
	authzKeeper := tApp.GetAuthzKeeper()
	err := bankKeeper.MintCoins(ctx, evmtypes.ModuleName, mintCoins)
	require.NoError(t, err)
	tApp.GetBankKeeper().SendCoinsFromModuleToAccount(ctx, evmtypes.ModuleName, granterAddr, grantCoins)
	tApp.GetBankKeeper().SendCoinsFromModuleToAccount(ctx, evmtypes.ModuleName, granteeAddr, grantCoins)
	tApp.GetBankKeeper().SetParams(ctx, banktypes.DefaultParams())

	// grant
	authorization := banktypes.NewSendAuthorization(grantCoins, []sdk.AccAddress{})
	setGrantMsg, err := authztypes.NewMsgGrant(granterAddr, granteeAddr, authorization, nil)
	require.NoError(t, err)

	_, err = authzKeeper.Grant(ctx, setGrantMsg)
	require.NoError(t, err)

	// set evm params
	EVMParams := tApp.GetEVMKeeper().GetParams(ctx)
	EVMParams.ActiveStaticPrecompiles = append(EVMParams.ActiveStaticPrecompiles, authz.AuthzContractAddress)
	tApp.GetEVMKeeper().SetParams(ctx, EVMParams)

	p, found, err := tApp.GetEVMKeeper().GetPrecompileInstance(ctx, common.HexToAddress(authz.AuthzContractAddress))
	require.True(t, found)
	require.NoError(t, err)

	contract := p.Map[common.HexToAddress(authz.AuthzContractAddress)]
	require.NotNil(t, contract)

	suppliedGas := uint64(20_000_000)
	execGrantMethod := authz.ABI.Methods[authz.ExecGrantMethod]

	// Create the EVM
	evm := vm.EVM{
		StateDB: statedb.New(ctx, tApp.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))),
	}

	args, err := execGrantMethod.Inputs.Pack(granterEvmAddr, recipientEvmAddr, denom, transferCoins[0].Amount.BigInt())
	require.Nil(t, err)
	res, _, err := evm.RunPrecompiledContract(
		contract,
		vm.AccountRef(granteeEvmAddr),
		append(execGrantMethod.ID, args...),
		suppliedGas,
		nil,
		false,
	)
	require.Nil(t, err)
	output, err := execGrantMethod.Outputs.Unpack(res)
	require.Nil(t, err)
	require.Equal(t, 1, len(output))
	require.Equal(t, output[0].(bool), true)

	// Because evm using cache context, we need to get the cache context
	cacheCtx, err := evm.StateDB.(*statedb.StateDB).GetCacheContext()
	require.NoError(t, err)

	granterBalance := bankKeeper.GetBalance(cacheCtx, granterAddr, denom)
	require.Equal(t, granterBalance, sdk.NewCoin(denom, grantCoins[0].Amount.Sub(transferCoins[0].Amount)))

	grantMsg := &authztypes.QueryGrantsRequest{
		Granter:    granterAddr.String(),
		Grantee:    granteeAddr.String(),
		MsgTypeUrl: banktypes.SendAuthorization{}.MsgTypeURL(),
		Pagination: nil,
	}

	grant, err := authzKeeper.Grants(cacheCtx, grantMsg)
	require.Nil(t, err)
	require.Equal(t, 1, len(grant.Grants))

	var sendAuthorization banktypes.SendAuthorization
	var grantCoin sdk.Coin

	for _, g := range grant.Grants {
		sendAuthorization.Unmarshal(g.Authorization.Value)

		for _, coin := range sendAuthorization.SpendLimit {
			if coin.Denom == denom {
				grantCoin = coin
			}
		}
	}

	require.Equal(t, grantCoin.Denom, denom)
	require.Equal(t, grantCoin.Amount, grantCoins[0].Amount.Sub(transferCoins[0].Amount))
}
