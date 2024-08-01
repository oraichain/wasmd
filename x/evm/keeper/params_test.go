package keeper_test

import (
	"github.com/CosmWasm/wasmd/x/evm/types"
)

func (suite *KeeperTestSuite) TestParams() {
	params := suite.app.EvmKeeper.GetParams(suite.ctx)
	params.EIP712AllowedMsgs = []types.EIP712AllowedMsg{}
	suite.Require().Equal(types.DefaultParams(), params)
	params.EvmDenom = "inj"
	suite.app.EvmKeeper.SetParams(suite.ctx, params)
	newParams := suite.app.EvmKeeper.GetParams(suite.ctx)
	newParams.EIP712AllowedMsgs = []types.EIP712AllowedMsg{}
	suite.Require().Equal(newParams, params)
}
