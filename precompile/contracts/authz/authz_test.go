package authz_test

import (
	"encoding/hex"
	"testing"

	"github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/precompile/contracts/authz"
	"github.com/CosmWasm/wasmd/precompile/registry"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/go-bip39"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/evmos/ethermint/x/evm/statedb"
	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
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
	denom := "ukava"
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

	evm := vm.EVM{
		StateDB: statedb.New(ctx, tApp.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))),
	}
	p := authz.NewContract(tApp.EvmKeeper, authzKeeper)
	method := authz.ABI.Methods[authz.SetGrantMethod]
	suppliedGas := uint64(10_000_000)

	args, err := method.Inputs.Pack(granteeEvmAddr, denom, grantCoins[0].Amount.BigInt())
	require.Nil(t, err)
	res, _, err := p.Run(&evm, granterEvmAddr, registry.AddrContractAddress,
		append(method.ID, args...),
		suppliedGas,
		false,
		nil,
	)
	require.Nil(t, err)
	output, err := method.Outputs.Unpack(res)
	require.Nil(t, err)
	require.Equal(t, 1, len(output))
	require.Equal(t, output[0].(bool), true)

	grantMsg := &authztypes.QueryGrantsRequest{
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
	require.Equal(t, grantCoin.Amount, grantCoins[0].Amount)
}
