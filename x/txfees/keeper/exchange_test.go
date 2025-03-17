package keeper_test

import (
	_ "embed"
	"fmt"

	"github.com/CosmWasm/wasmd/x/txfees/types"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

//go:embed testdata/mock-oraidex-v3.wasm
var mockOraidexv3Contract []byte

//go:embed testdata/price-query-local.wasm
var priceQueryContract []byte

func (s *KeeperTestSuite) TestQueryOraiDexTokenExchangeRate() {
	// setup test
	s.SetupTest()

	// setup mock contract for testing
	_ = s.SetupPriceContract()

	queryRate, err := s.feeKeeper.QueryOraiDexTokenExchangeRate(s.ctx, "factory/orai1wuvhex9xqs3r539mvc6mtm7n20fcj3qr2m0y9khx6n5vtlngfzes3k0rq9/DYeTA4ZQhEwoJ5imjq1Q3zgwfTgkh4WmdfFHAq3jLrv3")
	s.Require().NoError(err)
	s.Require().NotNil(queryRate)
}

func (s *KeeperTestSuite) SetupPriceContract() sdk.AccAddress {
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
	exeMsg := `{"add_pool": {"quote_token": "factory/orai1wuvhex9xqs3r539mvc6mtm7n20fcj3qr2m0y9khx6n5vtlngfzes3k0rq9/DYeTA4ZQhEwoJ5imjq1Q3zgwfTgkh4WmdfFHAq3jLrv3","fee_tier": {"fee": 3000000000, "tick_spacing": 100}}}`
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
	queryMsg := `{"get_sqrt_price": {"quote_token": "factory/orai1wuvhex9xqs3r539mvc6mtm7n20fcj3qr2m0y9khx6n5vtlngfzes3k0rq9/DYeTA4ZQhEwoJ5imjq1Q3zgwfTgkh4WmdfFHAq3jLrv3"}}`
	_, err = s.wasmKeeper.QuerySmart(s.ctx, priceContractAccAddress, []byte(queryMsg)) // "578941100392495845759830"
	s.Require().NoError(err)

	s.feeKeeper.SetParams(s.ctx, types.Params{
		TokenBaseDenom:       "orai",
		PriceContractAddress: instantiateResult.Address,
	})

	return priceContractAccAddress
}
