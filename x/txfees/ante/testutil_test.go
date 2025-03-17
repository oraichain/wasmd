package ante_test

import (
	_ "embed"
	"fmt"
	"testing"
	"time"

	"github.com/CosmWasm/wasmd/app"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	// TODO We don't need to import these API types if we use gogo's registry
	// ref: https://github.com/cosmos/cosmos-sdk/issues/14647
	_ "cosmossdk.io/api/cosmos/bank/v1beta1"
	_ "cosmossdk.io/api/cosmos/crypto/secp256k1"

	txfeeskeeper "github.com/CosmWasm/wasmd/x/txfees/keeper"
	"github.com/CosmWasm/wasmd/x/txfees/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/client"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/testutil"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/std"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	_ "github.com/cosmos/cosmos-sdk/testutil/testdata/testpb"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"

	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	txfeesante "github.com/CosmWasm/wasmd/x/txfees/ante"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
)

//go:embed testdata/mock-oraidex-v3.wasm
var mockOraidexv3Contract []byte

//go:embed testdata/price-query-local.wasm
var priceQueryContract []byte

// TestAccount represents an account used in the tests in x/auth/ante.
type TestAccount struct {
	acc  sdk.AccountI
	priv cryptotypes.PrivKey
}

// AnteTestSuite is a test suite to be used with ante handler tests.
type AnteTestSuite struct {
	suite.Suite

	anteHandler sdk.AnteHandler
	ctx         sdk.Context
	clientCtx   client.Context
	txBuilder   client.TxBuilder
	app         *app.WasmApp
	encCfg      TestEncodingConfig

	ak  authkeeper.AccountKeeper
	bk  bankkeeper.BaseKeeper
	fgk feegrantkeeper.Keeper
	tfk txfeeskeeper.Keeper
	wk  wasmkeeper.Keeper

	TestAccount TestAccount
}

type TestEncodingConfig struct {
	InterfaceRegistry cdctypes.InterfaceRegistry
	Codec             codec.Codec
	TxConfig          client.TxConfig
	Amino             *codec.LegacyAmino
}

func (s *AnteTestSuite) SetupTest() {
	s.app = app.Setup(s.T())
	s.ctx = s.app.NewContextLegacy(true, cmtproto.Header{
		Height:  1,
		Time:    time.Now(),
		ChainID: fmt.Sprintf("test-chain-%d", 1),
	})

	aminoCodec := codec.NewLegacyAmino()
	interfaceRegistry := testutil.CodecOptions{}.NewInterfaceRegistry()
	codec := codec.NewProtoCodec(interfaceRegistry)

	encCfg := TestEncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Codec:             codec,
		TxConfig:          tx.NewTxConfig(codec, tx.DefaultSignModes),
		Amino:             aminoCodec,
	}
	std.RegisterLegacyAminoCodec(encCfg.Amino)
	std.RegisterInterfaces(encCfg.InterfaceRegistry)
	s.app.BasicModuleManager.RegisterLegacyAminoCodec(encCfg.Amino)
	s.app.BasicModuleManager.RegisterInterfaces(encCfg.InterfaceRegistry)
	// register interfaces for testmsg
	encCfg.Amino.RegisterConcrete(&testdata.TestMsg{}, "testdata.TestMsg", nil)
	testdata.RegisterInterfaces(encCfg.InterfaceRegistry)

	s.ak = s.app.AccountKeeper
	s.bk = s.app.BankKeeper
	s.fgk = s.app.FeeGrantKeeper
	s.tfk = s.app.TxFeesKeeper
	s.wk = s.app.WasmKeeper

	s.encCfg = encCfg
	s.clientCtx = client.Context{}.
		WithTxConfig(s.encCfg.TxConfig).
		WithClient(clitestutil.NewMockCometRPC(abci.ResponseQuery{}))
	testAccount := s.CreateTestAccounts(1)
	s.TestAccount = testAccount[0]

	anteHandler := sdk.ChainAnteDecorators(
		txfeesante.NewMempoolFeeDecorator([]string{}, s.tfk),
		txfeesante.NewDeductFeeDecorator(s.ak, s.bk, s.fgk, s.tfk),
	)
	s.anteHandler = anteHandler

	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()
}

func TestAnteTestSuite(t *testing.T) {
	suite.Run(t, new(AnteTestSuite))
}

func (s *AnteTestSuite) CreateTestAccounts(numAccs int) []TestAccount {
	var accounts []TestAccount

	for i := 0; i < numAccs; i++ {
		priv, _, addr := testdata.KeyTestPubAddr()
		acc := s.ak.NewAccountWithAddress(s.ctx, addr)
		acc.SetAccountNumber(uint64(i + 1000))
		s.ak.SetAccount(s.ctx, acc)
		accounts = append(accounts, TestAccount{acc, priv})
	}

	return accounts
}

func FundAccount(ctx sdk.Context, bankKeeper bankkeeper.Keeper, addr sdk.AccAddress, amounts sdk.Coins) error {
	if err := bankKeeper.MintCoins(ctx, minttypes.ModuleName, amounts); err != nil {
		return err
	}

	return bankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, addr, amounts)
}

func (s *AnteTestSuite) CreateTestTx(privs []cryptotypes.PrivKey, accNums []uint64, accSeqs []uint64, chainID string) (xauthsigning.Tx, error) {
	var sigsV2 []signing.SignatureV2
	for i, priv := range privs {
		sigV2 := signing.SignatureV2{
			PubKey: priv.PubKey(),
			Data: &signing.SingleSignatureData{
				SignMode:  signing.SignMode(s.clientCtx.TxConfig.SignModeHandler().DefaultMode()),
				Signature: nil,
			},
			Sequence: accSeqs[i],
		}

		sigsV2 = append(sigsV2, sigV2)
	}

	if err := s.txBuilder.SetSignatures(sigsV2...); err != nil {
		return nil, err
	}

	sigsV2 = []signing.SignatureV2{}
	for i, priv := range privs {
		signerData := xauthsigning.SignerData{
			ChainID:       chainID,
			AccountNumber: accNums[i],
			Sequence:      accSeqs[i],
		}
		sigV2, err := clienttx.SignWithPrivKey(
			s.ctx,
			signing.SignMode(s.clientCtx.TxConfig.SignModeHandler().DefaultMode()),
			signerData,
			s.txBuilder,
			priv,
			s.clientCtx.TxConfig,
			accSeqs[i],
		)
		if err != nil {
			return nil, err
		}

		sigsV2 = append(sigsV2, sigV2)
	}

	if err := s.txBuilder.SetSignatures(sigsV2...); err != nil {
		return nil, err
	}

	return s.txBuilder.GetTx(), nil
}

func (s *AnteTestSuite) SetupPriceContract() sdk.AccAddress {
	_, _, sender := KeyTestPubAddr()
	msgStoreCode := wasmtypes.MsgStoreCodeFixture(func(m *wasmtypes.MsgStoreCode) {
		m.WASMByteCode = mockOraidexv3Contract
		m.Sender = sender.String()
	})

	// when
	rsp, err := s.app.MsgServiceRouter().Handler(msgStoreCode)(s.ctx, msgStoreCode)
	s.Require().NoError(err)
	var storeResult wasmtypes.MsgStoreCodeResponse
	s.Require().NoError(s.app.AppCodec().Unmarshal(rsp.Data, &storeResult))
	s.Require().Equal(uint64(1), storeResult.CodeID)

	// instantiate mock oraidex v3
	msgInstantiate := wasmtypes.MsgInstantiateContractFixture(func(m *wasmtypes.MsgInstantiateContract) {
		m.Sender = sender.String()
		m.Admin = sender.String()
		m.CodeID = 1
		m.Label = "mock-oraidex-v3"
		m.Msg = []byte(`{}`)
		m.Funds = sdk.Coins{}
	})

	// when
	rsp, err = s.app.MsgServiceRouter().Handler(msgInstantiate)(s.ctx, msgInstantiate)
	s.Require().NoError(err)
	var instantiateResult wasmtypes.MsgInstantiateContractResponse
	s.Require().NoError(s.app.AppCodec().Unmarshal(rsp.Data, &instantiateResult))

	// store price query contract
	msgStoreCode = wasmtypes.MsgStoreCodeFixture(func(m *wasmtypes.MsgStoreCode) {
		m.WASMByteCode = priceQueryContract
		m.Sender = sender.String()
	})

	// when
	rsp, err = s.app.MsgServiceRouter().Handler(msgStoreCode)(s.ctx, msgStoreCode)
	s.Require().NoError(err)
	s.Require().NoError(s.app.AppCodec().Unmarshal(rsp.Data, &storeResult))
	s.Require().Equal(uint64(2), storeResult.CodeID)

	// instantiate price query contract
	initMsg := fmt.Sprintf(`{"base_token": "orai", "oraidex_v3_addr": "%s"}`, instantiateResult.Address)
	msgInstantiate = wasmtypes.MsgInstantiateContractFixture(func(m *wasmtypes.MsgInstantiateContract) {
		m.Sender = sender.String()
		m.Admin = sender.String()
		m.CodeID = 2
		m.Label = "query-price-contract"
		m.Msg = []byte(initMsg)
		m.Funds = sdk.Coins{}
	})

	// when
	rsp, err = s.app.MsgServiceRouter().Handler(msgInstantiate)(s.ctx, msgInstantiate)
	s.Require().NoError(err)
	s.Require().NoError(s.app.AppCodec().Unmarshal(rsp.Data, &instantiateResult))

	priceContractAccAddress, err := sdk.AccAddressFromBech32(instantiateResult.Address)
	s.Require().NoError(err)
	// add USDAI test token pool
	exeMsg := `{"add_pool": {"quote_token": "usdai","fee_tier": {"fee": 3000000000, "tick_spacing": 100}}}`
	msgExecuteContract := wasmtypes.MsgExecuteContractFixture(func(m *wasmtypes.MsgExecuteContract) {
		m.Sender = sender.String()
		m.Contract = instantiateResult.Address
		m.Msg = []byte(exeMsg)
		m.Funds = sdk.Coins{}
	})

	// when
	_, err = s.app.MsgServiceRouter().Handler(msgExecuteContract)(s.ctx, msgExecuteContract)
	s.Require().NoError(err)

	// query
	queryMsg := `{"get_sqrt_price": {"quote_token": "usdai"}}`
	_, err = s.wk.QuerySmart(s.ctx, priceContractAccAddress, []byte(queryMsg)) // "578941100392495845759830"
	s.Require().NoError(err)

	s.tfk.SetParams(s.ctx, types.Params{
		TokenBaseDenom:       "orai",
		PriceContractAddress: instantiateResult.Address,
	})

	return priceContractAccAddress
}

func KeyTestPubAddr() (cryptotypes.PrivKey, cryptotypes.PubKey, sdk.AccAddress) {
	key := secp256k1.GenPrivKey()
	pub := key.PubKey()
	addr := sdk.AccAddress(pub.Address())
	return key, pub, addr
}
