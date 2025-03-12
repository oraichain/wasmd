package keeper_test

import (
	"testing"

	"github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/x/txfees/keeper"
	"github.com/CosmWasm/wasmd/x/txfees/types"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	"github.com/stretchr/testify/suite"
)

type KeeperTestSuite struct {
	suite.Suite

	ctx         sdk.Context
	app         *app.WasmApp
	feeKeeper   keeper.Keeper
	wasmKeeper  wasmkeeper.Keeper
	govKeeper   govkeeper.Keeper
	queryClient types.QueryClient
	msgServer   types.MsgServer
}

func (s *KeeperTestSuite) SetupTest() {
	s.app = app.Setup(s.T())
	s.ctx = s.app.NewContextLegacy(true, cmtproto.Header{Height: 1})

	s.feeKeeper = s.app.TxFeesKeeper
	s.wasmKeeper = s.app.WasmKeeper
	s.govKeeper = s.app.GovKeeper

	queryHelper := baseapp.NewQueryServerTestHelper(s.ctx, s.app.InterfaceRegistry())
	types.RegisterQueryServer(queryHelper, s.feeKeeper)
	s.queryClient = types.NewQueryClient(queryHelper)

	s.msgServer = keeper.NewMsgServerImpl(s.feeKeeper)
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) TestSetParams() {
	params := types.Params{
		PriceContractAddress: "testing",
	}

	// set params
	s.feeKeeper.SetParams(s.ctx, params)

	// get params from store
	storedParams, err := s.feeKeeper.GetParams(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(params, storedParams)
}
