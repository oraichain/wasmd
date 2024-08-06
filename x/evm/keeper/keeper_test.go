package keeper_test

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/CosmWasm/wasmd/app"
	feemarkettypes "github.com/CosmWasm/wasmd/x/feemarket/types"
	tmjson "github.com/cometbft/cometbft/libs/json"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simapp "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/CosmWasm/wasmd/crypto/ethsecp256k1"
	"github.com/CosmWasm/wasmd/tests"
	"github.com/CosmWasm/wasmd/x/evm/statedb"
	"github.com/CosmWasm/wasmd/x/evm/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/tmhash"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmversion "github.com/cometbft/cometbft/proto/tendermint/version"
	"github.com/cometbft/cometbft/version"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
)

var testTokens = sdkmath.NewIntWithDecimal(1000, 18)

type KeeperTestSuite struct {
	suite.Suite

	ctx         sdk.Context
	app         *app.WasmApp
	queryClient types.QueryClient
	address     common.Address
	consAddress sdk.ConsAddress

	// for generate test tx
	clientCtx client.Context
	ethSigner ethtypes.Signer

	appCodec codec.Codec
	signer   keyring.Signer

	enableFeemarket  bool
	enableLondonHF   bool
	mintFeeCollector bool
}

func (suite *KeeperTestSuite) EventsContains(events sdk.Events, expectedEvent sdk.Event) {
	foundMatch := false
	for _, event := range events {
		if event.Type == expectedEvent.Type {
			if reflect.DeepEqual(attrsToMap(expectedEvent.Attributes), attrsToMap(event.Attributes)) {
				foundMatch = true
			}
		}
	}

	suite.Truef(foundMatch, "event of type %s not found or did not match", expectedEvent.Type)
}

func attrsToMap(attrs []abci.EventAttribute) []sdk.Attribute {
	out := []sdk.Attribute{}

	for _, attr := range attrs {
		out = append(out, sdk.NewAttribute(string(attr.Key), string(attr.Value)))
	}

	return out
}

// GetEvents returns emitted events on the sdk context
func (suite *KeeperTestSuite) GetEvents() sdk.Events {
	return suite.ctx.EventManager().Events()
}

// / DoSetupTest setup test environment, it uses`require.TestingT` to support both `testing.T` and `testing.B`.
func (suite *KeeperTestSuite) DoSetupTest(t require.TestingT) {
	checkTx := false

	// account key, use a constant account to keep unit test deterministic.
	ecdsaPriv, err := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	require.NoError(t, err)
	priv := &ethsecp256k1.PrivKey{
		Key: crypto.FromECDSA(ecdsaPriv),
	}
	suite.address = common.BytesToAddress(priv.PubKey().Address().Bytes())
	suite.signer = tests.NewSigner(priv)

	// consensus key
	priv, err = ethsecp256k1.GenerateKey()
	require.NoError(t, err)
	suite.consAddress = sdk.ConsAddress(priv.PubKey().Address())
	suite.app = app.Setup(suite.T())
	genesis := app.GenesisStateWithSingleValidator(suite.T(), suite.app)

	feemarketGenesis := feemarkettypes.DefaultGenesisState()
	if suite.enableFeemarket {
		feemarketGenesis.Params.EnableHeight = 1
		feemarketGenesis.Params.NoBaseFee = false
	} else {
		feemarketGenesis.Params.NoBaseFee = true
	}
	genesis[feemarkettypes.ModuleName] = suite.app.AppCodec().MustMarshalJSON(feemarketGenesis)
	if !suite.enableLondonHF {
		evmGenesis := types.DefaultGenesisState()
		maxInt := sdkmath.NewInt(math.MaxInt64)
		evmGenesis.Params.ChainConfig.LondonBlock = &maxInt
		evmGenesis.Params.ChainConfig.ArrowGlacierBlock = &maxInt
		evmGenesis.Params.ChainConfig.MergeForkBlock = &maxInt
		genesis[types.ModuleName] = suite.app.AppCodec().MustMarshalJSON(evmGenesis)
	}

	if suite.mintFeeCollector {
		// mint some coin to fee collector
		coins := sdk.NewCoins(sdk.NewCoin(types.DefaultEVMDenom, sdkmath.NewInt(int64(params.TxGas)-1)))
		genesisState := suite.app.DefaultGenesis()
		balances := []banktypes.Balance{
			{
				Address: suite.app.AccountKeeper.GetModuleAddress(authtypes.FeeCollectorName).String(),
				Coins:   coins,
			},
		}
		// update total supply
		bankGenesis := banktypes.NewGenesisState(banktypes.DefaultGenesisState().Params, balances, sdk.NewCoins(sdk.NewCoin(types.DefaultEVMDenom, sdkmath.NewInt((int64(params.TxGas)-1)))), []banktypes.Metadata{}, nil)
		bz := suite.app.AppCodec().MustMarshalJSON(bankGenesis)
		require.NotNil(t, bz)
		genesisState[banktypes.ModuleName] = suite.app.AppCodec().MustMarshalJSON(bankGenesis)

		// we marshal the genesisState of all module to a byte array
		stateBytes, err := tmjson.MarshalIndent(genesisState, "", " ")
		require.NoError(t, err)

		// Initialize the chain
		suite.app.InitChain(
			&abci.RequestInitChain{
				ChainId:         "ethermint_9000-1",
				Validators:      []abci.ValidatorUpdate{},
				ConsensusParams: simapp.DefaultConsensusParams,
				AppStateBytes:   stateBytes,
			},
		)
	}

	suite.ctx = suite.app.BaseApp.NewContextLegacy(checkTx, tmproto.Header{
		Height:          1,
		ChainID:         "ethermint_9000-1",
		Time:            time.Now().UTC(),
		ProposerAddress: suite.consAddress.Bytes(),
		Version: tmversion.Consensus{
			Block: version.BlockProtocol,
		},
		LastBlockId: tmproto.BlockID{
			Hash: tmhash.Sum([]byte("block_id")),
			PartSetHeader: tmproto.PartSetHeader{
				Total: 11,
				Hash:  tmhash.Sum([]byte("partset_header")),
			},
		},
		AppHash:            tmhash.Sum([]byte("app")),
		DataHash:           tmhash.Sum([]byte("data")),
		EvidenceHash:       tmhash.Sum([]byte("evidence")),
		ValidatorsHash:     tmhash.Sum([]byte("validators")),
		NextValidatorsHash: tmhash.Sum([]byte("next_validators")),
		ConsensusHash:      tmhash.Sum([]byte("consensus")),
		LastResultsHash:    tmhash.Sum([]byte("last_result")),
	})

	queryHelper := baseapp.NewQueryServerTestHelper(suite.ctx, suite.app.InterfaceRegistry())
	types.RegisterQueryServer(queryHelper, suite.app.EvmKeeper)
	suite.queryClient = types.NewQueryClient(queryHelper)

	// acc := &apptypes.EthAccount{
	// 	BaseAccount: authtypes.NewBaseAccount(sdk.AccAddress(suite.address.Bytes()), nil, 0, 0),
	// 	CodeHash:    common.BytesToHash(crypto.Keccak256(nil)).String(),
	// }
	acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, suite.address.Bytes())

	suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

	valAddr := sdk.ValAddress(suite.address.Bytes())
	validator, err := stakingtypes.NewValidator(valAddr.String(), priv.PubKey(), stakingtypes.Description{})
	require.NoError(t, err)

	err = suite.app.StakingKeeper.SetValidatorByConsAddr(suite.ctx, validator)
	require.NoError(t, err)
	err = suite.app.StakingKeeper.SetValidator(suite.ctx, validator)
	require.NoError(t, err)

	encodingConfig := app.MakeEncodingConfig(suite.T())
	suite.clientCtx = client.Context{}.WithTxConfig(encodingConfig.TxConfig)
	suite.ethSigner = ethtypes.LatestSignerForChainID(suite.app.EvmKeeper.ChainID())
	suite.appCodec = encodingConfig.Codec
}

func (suite *KeeperTestSuite) SetupTest() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("orai", "oraipub")
	suite.DoSetupTest(suite.T())
}

func (suite *KeeperTestSuite) EvmDenom() string {
	ctx := sdk.WrapSDKContext(suite.ctx)
	rsp, _ := suite.queryClient.Params(ctx, &types.QueryParamsRequest{})
	return rsp.Params.EvmDenom
}

// Commit and begin new block
func (suite *KeeperTestSuite) Commit() {
	_, _ = suite.app.Commit()
	header := suite.ctx.BlockHeader()
	header.Height += 1
	suite.app.BeginBlocker(suite.ctx)

	// update ctx
	suite.ctx = suite.app.BaseApp.NewContextLegacy(false, header)

	queryHelper := baseapp.NewQueryServerTestHelper(suite.ctx, suite.app.InterfaceRegistry())
	types.RegisterQueryServer(queryHelper, suite.app.EvmKeeper)
	suite.queryClient = types.NewQueryClient(queryHelper)
}

func (suite *KeeperTestSuite) StateDB() *statedb.StateDB {
	return statedb.New(suite.ctx, suite.app.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(suite.ctx.HeaderHash())))
}

// DeployTestContract deploy a test erc20 contract and returns the contract address
func (suite *KeeperTestSuite) DeployTestContract(t require.TestingT, owner common.Address, supply *big.Int) common.Address {
	ctx := sdk.WrapSDKContext(suite.ctx)
	chainID := suite.app.EvmKeeper.ChainID()

	ctorArgs, err := types.ERC20Contract.ABI.Pack("", owner, supply)
	require.NoError(t, err)

	nonce := suite.app.EvmKeeper.GetNonce(suite.ctx, suite.address)

	data := append(types.ERC20Contract.Bin, ctorArgs...)
	args, err := json.Marshal(&types.TransactionArgs{
		From: &suite.address,
		Data: (*hexutil.Bytes)(&data),
	})
	require.NoError(t, err)

	res, err := suite.queryClient.EstimateGas(ctx, &types.EthCallRequest{
		Args:   args,
		GasCap: uint64(types.DefaultGasCap),
	})
	require.NoError(t, err)

	var erc20DeployTx *types.MsgEthereumTx
	if suite.enableFeemarket {
		erc20DeployTx = types.NewTxContract(
			chainID,
			nonce,
			nil,     // amount
			res.Gas, // gasLimit
			nil,     // gasPrice
			suite.app.FeeMarketKeeper.GetBaseFee(suite.ctx),
			big.NewInt(1),
			data,                   // input
			&ethtypes.AccessList{}, // accesses
		)
	} else {
		erc20DeployTx = types.NewTxContract(
			chainID,
			nonce,
			nil,     // amount
			res.Gas, // gasLimit
			nil,     // gasPrice
			nil, nil,
			data, // input
			nil,  // accesses
		)
	}

	erc20DeployTx.From = suite.address.Hex()
	err = erc20DeployTx.Sign(ethtypes.LatestSignerForChainID(chainID), suite.signer)
	require.NoError(t, err)
	rsp, err := suite.app.EvmKeeper.EthereumTx(ctx, erc20DeployTx)
	require.NoError(t, err)
	require.Empty(t, rsp.VmError)
	return crypto.CreateAddress(suite.address, nonce)
}

func (suite *KeeperTestSuite) TransferERC20Token(t require.TestingT, contractAddr, from, to common.Address, amount *big.Int) *types.MsgEthereumTx {
	ctx := sdk.WrapSDKContext(suite.ctx)
	chainID := suite.app.EvmKeeper.ChainID()

	transferData, err := types.ERC20Contract.ABI.Pack("transfer", to, amount)
	require.NoError(t, err)
	args, err := json.Marshal(&types.TransactionArgs{To: &contractAddr, From: &from, Data: (*hexutil.Bytes)(&transferData)})
	require.NoError(t, err)
	res, err := suite.queryClient.EstimateGas(ctx, &types.EthCallRequest{
		Args:   args,
		GasCap: 25_000_000,
	})
	require.NoError(t, err)

	nonce := suite.app.EvmKeeper.GetNonce(suite.ctx, suite.address)

	var ercTransferTx *types.MsgEthereumTx
	if suite.enableFeemarket {
		ercTransferTx = types.NewTx(
			chainID,
			nonce,
			&contractAddr,
			nil,
			res.Gas,
			nil,
			suite.app.FeeMarketKeeper.GetBaseFee(suite.ctx),
			big.NewInt(1),
			transferData,
			&ethtypes.AccessList{}, // accesses
		)
	} else {
		ercTransferTx = types.NewTx(
			chainID,
			nonce,
			&contractAddr,
			nil,
			res.Gas,
			nil,
			nil, nil,
			transferData,
			nil,
		)
	}

	ercTransferTx.From = suite.address.Hex()
	err = ercTransferTx.Sign(ethtypes.LatestSignerForChainID(chainID), suite.signer)
	require.NoError(t, err)
	rsp, err := suite.app.EvmKeeper.EthereumTx(ctx, ercTransferTx)
	require.NoError(t, err)
	require.Empty(t, rsp.VmError)
	return ercTransferTx
}

// DeployTestMessageCall deploy a test erc20 contract and returns the contract address
func (suite *KeeperTestSuite) DeployTestMessageCall(t require.TestingT) common.Address {
	ctx := sdk.WrapSDKContext(suite.ctx)
	chainID := suite.app.EvmKeeper.ChainID()

	data := types.TestMessageCall.Bin
	args, err := json.Marshal(&types.TransactionArgs{
		From: &suite.address,
		Data: (*hexutil.Bytes)(&data),
	})
	require.NoError(t, err)

	res, err := suite.queryClient.EstimateGas(ctx, &types.EthCallRequest{
		Args:   args,
		GasCap: uint64(types.DefaultGasCap),
	})
	require.NoError(t, err)

	nonce := suite.app.EvmKeeper.GetNonce(suite.ctx, suite.address)

	var erc20DeployTx *types.MsgEthereumTx
	if suite.enableFeemarket {
		erc20DeployTx = types.NewTxContract(
			chainID,
			nonce,
			nil,     // amount
			res.Gas, // gasLimit
			nil,     // gasPrice
			suite.app.FeeMarketKeeper.GetBaseFee(suite.ctx),
			big.NewInt(1),
			data,                   // input
			&ethtypes.AccessList{}, // accesses
		)
	} else {
		erc20DeployTx = types.NewTxContract(
			chainID,
			nonce,
			nil,     // amount
			res.Gas, // gasLimit
			nil,     // gasPrice
			nil, nil,
			data, // input
			nil,  // accesses
		)
	}

	erc20DeployTx.From = suite.address.Hex()
	err = erc20DeployTx.Sign(ethtypes.LatestSignerForChainID(chainID), suite.signer)
	require.NoError(t, err)
	rsp, err := suite.app.EvmKeeper.EthereumTx(ctx, erc20DeployTx)
	require.NoError(t, err)
	require.Empty(t, rsp.VmError)
	return crypto.CreateAddress(suite.address, nonce)
}

func (suite *KeeperTestSuite) TestBaseFee() {
	testCases := []struct {
		name            string
		enableLondonHF  bool
		enableFeemarket bool
		expectBaseFee   *big.Int
	}{
		{"not enable london HF, not enable feemarket", false, false, nil},
		{"enable london HF, not enable feemarket", true, false, big.NewInt(0)},
		{"enable london HF, enable feemarket", true, true, big.NewInt(1000000000)},
		{"not enable london HF, enable feemarket", false, true, nil},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.enableFeemarket = tc.enableFeemarket
			suite.enableLondonHF = tc.enableLondonHF
			suite.SetupTest()
			suite.app.EvmKeeper.BeginBlock(suite.ctx)
			params := suite.app.EvmKeeper.GetParams(suite.ctx)
			ethCfg := params.ChainConfig.EthereumConfig(suite.app.EvmKeeper.ChainID())
			baseFee := suite.app.EvmKeeper.BaseFee(suite.ctx, ethCfg)
			suite.Require().Equal(tc.expectBaseFee, baseFee)
		})
	}
	suite.enableFeemarket = false
	suite.enableLondonHF = true
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, &KeeperTestSuite{
		enableFeemarket: false,
		enableLondonHF:  true,
	})
}

func (suite *KeeperTestSuite) TestMsgSetMappingEvmAddress() {
	signer := "orai1knzg7jdc49ghnc2pkqg6vks8ccsk6efzfgv6gv"
	pubkey := "AvSl0d9JrHCW4mdEyHvZu076WxLgH0bBVLigUcFm4UjV"
	expectedEvmAddress, _ := types.PubkeyToEVMAddress(pubkey)
	castAddress := sdk.AccAddress(expectedEvmAddress[:])
	acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, castAddress)
	acc.SetSequence(0)
	suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

	// fixture for migrate nonce
	signerAddress, _ := sdk.AccAddressFromBech32(signer)
	signerAcc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, signerAddress)
	signerAcc.SetSequence(1)
	suite.app.AccountKeeper.SetAccount(suite.ctx, signerAcc)

	// fixture for migrate balance
	mintCoins := sdk.NewCoins(sdk.NewCoin(suite.EvmDenom(), sdkmath.NewInt(50)))
	suite.app.BankKeeper.MintCoins(suite.ctx, types.ModuleName, mintCoins)
	sentCoins := sdk.NewCoins(sdk.NewCoin(suite.EvmDenom(), sdkmath.NewInt(5)))
	moduleAcc := suite.app.AccountKeeper.GetModuleAccount(suite.ctx, types.ModuleName)
	suite.app.BankKeeper.SendCoins(suite.ctx, moduleAcc.GetAddress(), castAddress, sentCoins)
	suite.app.BankKeeper.SendCoins(suite.ctx, moduleAcc.GetAddress(), signerAddress, sentCoins)

	type errArgs struct {
		expectPass bool
		contains   string
	}

	tests := []struct {
		name     string
		msg      types.MsgSetMappingEvmAddress
		errArgs  errArgs
		malleate func()
	}{
		{
			"valid",
			types.NewMsgSetMappingEvmAddress(
				signer,
				pubkey,
			),
			errArgs{
				expectPass: true,
			},
			func() {},
		},
		{
			"invalid - invalid signer",
			types.NewMsgSetMappingEvmAddress(
				"foobar",
				pubkey,
			),
			errArgs{
				expectPass: false,
				contains:   "invalid signer address",
			},
			func() {},
		},
		{
			"invalid - invalid pubkey",
			types.NewMsgSetMappingEvmAddress(
				signer,
				"Avalv/HkKw5oBST0LP6Hb8v+kLX22/V97IndXM2O6GeZ",
			),
			errArgs{
				expectPass: false,
				contains:   "Signer does not match the given pubkey",
			},
			func() {},
		},
		{
			"valid with migrate nonce",
			types.NewMsgSetMappingEvmAddress(
				signer,
				pubkey,
			),
			errArgs{
				expectPass: true,
			},
			func() {
				acc.SetSequence(10)
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
			},
		},
		{
			"valid with migrate balance",
			types.NewMsgSetMappingEvmAddress(
				signer,
				pubkey,
			),
			errArgs{
				expectPass: true,
			},
			func() {
				sentCoins := sdk.NewCoins(sdk.NewCoin(suite.EvmDenom(), sdkmath.NewInt(20)))
				suite.app.BankKeeper.SendCoins(suite.ctx, moduleAcc.GetAddress(), castAddress, sentCoins)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			tc.malleate()
			_, err := suite.app.EvmKeeper.SetMappingEvmAddress(sdk.WrapSDKContext(suite.ctx), &tc.msg)

			if tc.errArgs.expectPass {
				suite.Require().NoError(err)

				// validate user coin balance
				cosmosAccAddress := sdk.MustAccAddressFromBech32(signer)
				actualEvmAddress, _ := suite.app.EvmKeeper.GetEvmAddressMapping(suite.ctx, cosmosAccAddress)
				suite.Require().Equal(expectedEvmAddress.Hex(), actualEvmAddress.Hex(), "evm addresses dont match")

				// validate migrate nonce
				acc := suite.app.AccountKeeper.GetAccount(suite.ctx, castAddress)
				signerAcc := suite.app.AccountKeeper.GetAccount(suite.ctx, signerAddress)
				nonce := acc.GetSequence()
				signerNonce := signerAcc.GetSequence()
				suite.Require().GreaterOrEqual(signerNonce, nonce)

				// validate migrate balance
				castBalance := suite.app.BankKeeper.GetBalance(suite.ctx, castAddress, suite.EvmDenom())
				signerBalance := suite.app.BankKeeper.GetBalance(suite.ctx, signerAddress, suite.EvmDenom())
				fmt.Println("signer balance: ", signerBalance)
				suite.Require().GreaterOrEqual(signerBalance.Amount.Int64(), castBalance.Amount.Int64())
				if signerBalance.Amount.GT(castBalance.Amount) {
					suite.Require().Equal(castBalance.Amount.Int64(), int64(0))
				}

				// msg server event
				suite.EventsContains(suite.GetEvents(),
					sdk.NewEvent(
						sdk.EventTypeMessage,
						sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
						sdk.NewAttribute(sdk.AttributeKeySender, signer),
					))

				// keeper event
				suite.EventsContains(suite.GetEvents(),
					sdk.NewEvent(
						types.EventTypeSetMappingEvmAddress,
						sdk.NewAttribute(types.AttributeKeyCosmosAddress, signer),
						sdk.NewAttribute(types.AttributeKeyEvmAddress, actualEvmAddress.Hex()),
						sdk.NewAttribute(types.AttributeKeyPubkey, pubkey),
					))
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.errArgs.contains)
			}
			suite.app.EvmKeeper.DeleteAddressMapping(suite.ctx, signerAddress, *expectedEvmAddress)
		})
	}
}

func (suite *KeeperTestSuite) TestGetAccAddressBytesFromPubkey() {
	pubkeyString := "Ah4NweWyFaVG5xcOwY5I7Tm4mmfPgLtS+Qn3jvXLX0VP"
	compressedPubkeyBytes, _ := base64.StdEncoding.DecodeString(pubkeyString)
	ethPubkey := ethsecp256k1.PubKey{Key: compressedPubkeyBytes}
	cosmosPubkey := secp256k1.PubKey{Key: compressedPubkeyBytes}
	cosmosAddress := sdk.AccAddress(cosmosPubkey.Address().Bytes())
	cosmosAddressFromEvm := sdk.AccAddress(ethPubkey.Address().Bytes())
	evmAddress := common.BytesToAddress(ethPubkey.Address().Bytes())

	type errArgs struct {
		expectPass bool
		contains   string
	}

	tests := []struct {
		name               string
		errArgs            errArgs
		pubkey             cryptotypes.PubKey
		expectedAccAddress string
		malleate           func()
	}{
		{
			"secp256k1 pubkey valid",
			errArgs{
				expectPass: true,
			},
			&cosmosPubkey,
			cosmosAddress.String(),
			func() {},
		},
		{
			"eth_secp256k1 pubkey valid with no address mapping",
			errArgs{
				expectPass: true,
			},
			&ethPubkey,
			cosmosAddressFromEvm.String(),
			func() {},
		},
		{
			"eth_secp256k1 pubkey valid with addess mapping",
			errArgs{
				expectPass: true,
			},
			&ethPubkey,
			cosmosAddress.String(),
			func() {
				suite.app.EvmKeeper.SetAddressMapping(suite.ctx, cosmosAddress, evmAddress)
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			tc.malleate()
			accAddress, err := suite.app.EvmKeeper.GetAccAddressBytesFromPubkey(suite.ctx, tc.pubkey)

			if tc.errArgs.expectPass {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expectedAccAddress, sdk.AccAddress(accAddress).String())
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.errArgs.contains)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestValidateSignerEIP712Ante() {
	pubkeyString := "Ah4NweWyFaVG5xcOwY5I7Tm4mmfPgLtS+Qn3jvXLX0VP"
	compressedPubkeyBytes, _ := base64.StdEncoding.DecodeString(pubkeyString)
	ethPubkey := ethsecp256k1.PubKey{Key: compressedPubkeyBytes}
	cosmosPubkey := secp256k1.PubKey{Key: compressedPubkeyBytes}
	cosmosAddress := sdk.AccAddress(cosmosPubkey.Address().Bytes())
	cosmosAddressFromEvm := sdk.AccAddress(ethPubkey.Address().Bytes())
	evmAddress := common.BytesToAddress(ethPubkey.Address().Bytes())

	type errArgs struct {
		expectPass bool
		contains   string
	}

	tests := []struct {
		name     string
		errArgs  errArgs
		pubkey   cryptotypes.PubKey
		signer   sdk.AccAddress
		malleate func()
	}{
		{
			"secp256k1 pubkey valid",
			errArgs{
				expectPass: true,
			},
			&cosmosPubkey,
			cosmosAddress,
			func() {},
		},
		{
			"eth_secp256k1 pubkey valid with no address mapping",
			errArgs{
				expectPass: true,
			},
			&ethPubkey,
			cosmosAddressFromEvm,
			func() {},
		},
		{
			"eth_secp256k1 pubkey valid with addess mapping",
			errArgs{
				expectPass: true,
			},
			&ethPubkey,
			cosmosAddress,
			func() {
				suite.app.EvmKeeper.SetAddressMapping(suite.ctx, cosmosAddress, evmAddress)
			},
		},
		{
			"secp256k1 pubkey invalid signer don't match",
			errArgs{
				expectPass: false,
				contains:   "does not match signer",
			},
			&cosmosPubkey,
			cosmosAddressFromEvm,
			func() {
			},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			suite.SetupTest()
			tc.malleate()
			err := suite.app.EvmKeeper.ValidateSignerEIP712Ante(suite.ctx, tc.pubkey, tc.signer)

			if tc.errArgs.expectPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.errArgs.contains)
			}
		})
	}
}
