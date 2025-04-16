package bank_test

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/precompile/contracts/bank"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/evm/x/vm/core/vm"
	"github.com/cosmos/evm/x/vm/statedb"
	"github.com/cosmos/go-bip39"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
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

func TestSend(t *testing.T) {
	denom := "orai"
	tApp := app.Setup(t)
	ctx := tApp.NewContext(true)
	mockAddr, mockEVMAddr := MockAddressPair()
	sdk.RegisterDenom(denom, sdkmath.LegacyNewDec(6))
	receiveCosmosAddr, mockReceiverEVMAddr := MockAddressPair()
	tApp.EvmKeeper.SetAddressMapping(ctx, mockAddr, mockEVMAddr)
	tApp.EvmKeeper.SetAddressMapping(ctx, receiveCosmosAddr, mockReceiverEVMAddr)
	mintCoins := sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewInt(100000)))
	sentCoins := sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewInt(10)))
	bankKeeper := tApp.GetBankKeeper()
	err := bankKeeper.MintCoins(ctx, evmtypes.ModuleName, mintCoins)
	require.NoError(t, err)
	tApp.GetBankKeeper().SendCoinsFromModuleToAccount(ctx, evmtypes.ModuleName, mockAddr, sentCoins)
	tApp.GetBankKeeper().SetParams(ctx, banktypes.DefaultParams())

	// Set EVM parameters to register the bank precompile address
	EVMParams := tApp.GetEVMKeeper().GetParams(ctx)
	EVMParams.ActiveStaticPrecompiles = append(EVMParams.ActiveStaticPrecompiles, bank.BankPrecompileAddress)
	tApp.GetEVMKeeper().SetParams(ctx, EVMParams)
	p, found, err := tApp.GetEVMKeeper().GetPrecompileInstance(ctx, common.HexToAddress(bank.BankPrecompileAddress))

	require.True(t, found)
	require.NoError(t, err)

	contract := p.Map[common.HexToAddress(bank.BankPrecompileAddress)]
	require.NotNil(t, contract)

	evm := vm.EVM{
		StateDB: statedb.New(ctx, tApp.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))),
	}
	method := bank.ABI.Methods[bank.SendMethod]
	suppliedGas := uint64(10_000_000)

	args, err := method.Inputs.Pack(mockReceiverEVMAddr, denom, sentCoins[0].Amount.BigInt())
	require.Nil(t, err)
	res, _, err := evm.RunPrecompiledContract(
		contract,
		vm.AccountRef(mockEVMAddr),
		append(method.ID, args...),
		suppliedGas,
		nil,
		false,
	)
	require.Nil(t, err)
	output, err := method.Outputs.Unpack(res)
	require.Nil(t, err)
	require.Equal(t, 1, len(output))
	require.Equal(t, output[0].(bool), true)

	// commit the stateDB to get the updated context
	stateDB := evm.StateDB.(*statedb.StateDB)
	stateDB.Commit()

	balance := bankKeeper.GetBalance(ctx, receiveCosmosAddr, denom)
	require.Equal(t, balance.Amount, sdkmath.NewInt(10))
}

func TestBurn(t *testing.T) {
	denom := "orai"
	tApp := app.Setup(t)
	ctx := tApp.NewContext(true)
	sdk.RegisterDenom(denom, sdkmath.LegacyNewDec(6))

	burnCosmosAddr, burnEvmAddr := MockAddressPair()
	tApp.EvmKeeper.SetAddressMapping(ctx, burnCosmosAddr, burnEvmAddr)

	mintCoins := sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewInt(10000)))
	sentCoins := sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewInt(100)))
	burnCoins := sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewInt(10)))
	bankKeeper := tApp.GetBankKeeper()
	err := bankKeeper.MintCoins(ctx, evmtypes.ModuleName, mintCoins)
	require.NoError(t, err)
	tApp.GetBankKeeper().SendCoinsFromModuleToAccount(ctx, evmtypes.ModuleName, burnCosmosAddr, sentCoins)
	tApp.GetBankKeeper().SetParams(ctx, banktypes.DefaultParams())

	// Set EVM parameters to register the bank precompile address
	EVMParams := tApp.GetEVMKeeper().GetParams(ctx)
	EVMParams.ActiveStaticPrecompiles = append(EVMParams.ActiveStaticPrecompiles, bank.BankPrecompileAddress)
	tApp.GetEVMKeeper().SetParams(ctx, EVMParams)
	p, found, err := tApp.GetEVMKeeper().GetPrecompileInstance(ctx, common.HexToAddress(bank.BankPrecompileAddress))

	require.True(t, found)
	require.NoError(t, err)

	contract := p.Map[common.HexToAddress(bank.BankPrecompileAddress)]
	require.NotNil(t, contract)

	evm := vm.EVM{
		StateDB: statedb.New(ctx, tApp.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))),
	}
	method := bank.ABI.Methods[bank.BurnMethod]
	suppliedGas := uint64(10_000_000)

	args, err := method.Inputs.Pack(burnEvmAddr, denom, burnCoins[0].Amount.BigInt())
	require.Nil(t, err)

	res, _, err := evm.RunPrecompiledContract(
		contract,
		vm.AccountRef(burnEvmAddr),
		append(method.ID, args...),
		suppliedGas,
		nil,
		false,
	)
	require.Nil(t, err)
	output, err := method.Outputs.Unpack(res)
	require.Nil(t, err)
	require.Equal(t, 1, len(output))
	require.Equal(t, output[0].(bool), true)

	// commit the stateDB to get the updated context
	stateDB := evm.StateDB.(*statedb.StateDB)
	stateDB.Commit()

	balance := bankKeeper.GetBalance(ctx, burnCosmosAddr, denom)
	require.Equal(t, balance.Amount, sentCoins[0].Amount.Sub(burnCoins[0].Amount))
}

func TestBurnFrom(t *testing.T) {
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
	burnCoins := sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewInt(10)))
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
	setGrantMsg, err := authz.NewMsgGrant(granterAddr, granteeAddr, authorization, nil)
	require.NoError(t, err)

	_, err = authzKeeper.Grant(ctx, setGrantMsg)
	require.NoError(t, err)

	EVMParams := tApp.GetEVMKeeper().GetParams(ctx)
	EVMParams.ActiveStaticPrecompiles = append(EVMParams.ActiveStaticPrecompiles, bank.BankPrecompileAddress)
	tApp.GetEVMKeeper().SetParams(ctx, EVMParams)
	p, found, err := tApp.GetEVMKeeper().GetPrecompileInstance(ctx, common.HexToAddress(bank.BankPrecompileAddress))

	require.True(t, found)
	require.NoError(t, err)

	contract := p.Map[common.HexToAddress(bank.BankPrecompileAddress)]
	require.NotNil(t, contract)

	// burn from
	evm := vm.EVM{
		StateDB: statedb.New(ctx, tApp.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))),
	}
	method := bank.ABI.Methods[bank.BurnMethod]
	suppliedGas := uint64(10_000_000)

	args, err := method.Inputs.Pack(granterEvmAddr, denom, burnCoins[0].Amount.BigInt())
	require.Nil(t, err)
	res, _, err := evm.RunPrecompiledContract(
		contract,
		vm.AccountRef(granteeEvmAddr),
		append(method.ID, args...),
		suppliedGas,
		nil,
		false,
	)
	require.Nil(t, err)
	output, err := method.Outputs.Unpack(res)
	require.Nil(t, err)
	require.Equal(t, 1, len(output))
	require.Equal(t, output[0].(bool), true)

	// commit the stateDB to get the updated context
	stateDB := evm.StateDB.(*statedb.StateDB)
	stateDB.Commit()

	// query balance of granter after burn
	balance := bankKeeper.GetBalance(ctx, granterAddr, denom)
	require.Equal(t, balance.Amount, grantCoins[0].Amount.Sub(burnCoins[0].Amount))

	// query grant of grantee
	grantMsg := &authz.QueryGrantsRequest{
		Granter:    granterAddr.String(),
		Grantee:    granteeAddr.String(),
		MsgTypeUrl: banktypes.SendAuthorization{}.MsgTypeURL(),
		Pagination: nil,
	}

	grant, err := authzKeeper.Grants(ctx, grantMsg)
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
	require.Equal(t, grantCoin.Amount, grantCoins[0].Amount.Sub(burnCoins[0].Amount))
}

func TestBalance(t *testing.T) {
	denom := "orai"
	tApp := app.Setup(t)
	ctx := tApp.NewContext(true)
	mockAddr, mockEVMAddr := MockAddressPair()
	tApp.EvmKeeper.SetAddressMapping(ctx, mockAddr, mockEVMAddr)
	sdk.RegisterDenom(denom, sdkmath.LegacyNewDec(6))
	mintCoins := sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewInt(100000)))
	sentCoins := sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewInt(10)))
	bankKeeper := tApp.GetBankKeeper()
	err := bankKeeper.MintCoins(ctx, evmtypes.ModuleName, mintCoins)
	require.NoError(t, err)
	tApp.GetBankKeeper().SendCoinsFromModuleToAccount(ctx, evmtypes.ModuleName, mockAddr, sentCoins)
	tApp.GetBankKeeper().SetParams(ctx, banktypes.DefaultParams())

	// Set EVM parameters to register the bank precompile address
	EVMParams := tApp.GetEVMKeeper().GetParams(ctx)
	EVMParams.ActiveStaticPrecompiles = append(EVMParams.ActiveStaticPrecompiles, bank.BankPrecompileAddress)
	tApp.GetEVMKeeper().SetParams(ctx, EVMParams)
	p, found, err := tApp.GetEVMKeeper().GetPrecompileInstance(ctx, common.HexToAddress(bank.BankPrecompileAddress))

	require.True(t, found)
	require.NoError(t, err)

	contract := p.Map[common.HexToAddress(bank.BankPrecompileAddress)]
	require.NotNil(t, contract)

	evm := vm.EVM{
		StateDB: statedb.New(ctx, tApp.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))),
	}
	method := bank.ABI.Methods[bank.BalanceMethod]
	suppliedGas := uint64(10_000_000)

	args, err := method.Inputs.Pack(mockEVMAddr, denom)
	require.Nil(t, err)
	res, _, err := evm.RunPrecompiledContract(
		contract,
		vm.AccountRef(mockEVMAddr),
		append(method.ID, args...),
		suppliedGas,
		nil,
		false,
	)
	require.Nil(t, err)
	output, err := method.Outputs.Unpack(res)
	require.Nil(t, err)
	require.Equal(t, 1, len(output))
	require.Equal(t, output[0].(*big.Int), big.NewInt(sentCoins[0].Amount.Int64()))
}

func TestSupply(t *testing.T) {
	denom := "orai"
	tApp := app.Setup(t)
	ctx := tApp.NewContext(true)
	mockAddr, mockEVMAddr := MockAddressPair()
	tApp.EvmKeeper.SetAddressMapping(ctx, mockAddr, mockEVMAddr)
	sdk.RegisterDenom(denom, sdkmath.LegacyNewDec(6))
	mintCoins := sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewInt(100000)))
	bankKeeper := tApp.GetBankKeeper()
	err := bankKeeper.MintCoins(ctx, evmtypes.ModuleName, mintCoins)
	require.NoError(t, err)
	tApp.GetBankKeeper().SetParams(ctx, banktypes.DefaultParams())

	// Set EVM parameters to register the bank precompile address
	EVMParams := tApp.GetEVMKeeper().GetParams(ctx)
	EVMParams.ActiveStaticPrecompiles = append(EVMParams.ActiveStaticPrecompiles, bank.BankPrecompileAddress)
	tApp.GetEVMKeeper().SetParams(ctx, EVMParams)
	p, found, err := tApp.GetEVMKeeper().GetPrecompileInstance(ctx, common.HexToAddress(bank.BankPrecompileAddress))

	require.True(t, found)
	require.NoError(t, err)

	contract := p.Map[common.HexToAddress(bank.BankPrecompileAddress)]

	evm := vm.EVM{
		StateDB: statedb.New(ctx, tApp.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))),
	}
	method := bank.ABI.Methods[bank.SupplyMethod]
	suppliedGas := uint64(10_000_000)

	args, err := method.Inputs.Pack(denom)
	require.Nil(t, err)
	res, _, err := evm.RunPrecompiledContract(
		contract,
		vm.AccountRef(mockEVMAddr),
		append(method.ID, args...),
		suppliedGas,
		nil,
		false,
	)
	require.Nil(t, err)
	output, err := method.Outputs.Unpack(res)
	require.Nil(t, err)
	require.Equal(t, 1, len(output))
	require.Equal(t, output[0].(*big.Int), big.NewInt(mintCoins[0].Amount.Int64()))
}
