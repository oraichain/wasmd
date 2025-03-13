package keeper_test

import (
	"github.com/CosmWasm/wasmd/x/txfees/types"
)

func (s *KeeperTestSuite) TestMsgAddFeeToken() {
	testCases := []struct {
		name     string
		malleate func()
		msg      *types.MsgAddFeeToken
		expErr   bool
		assert   func(s *KeeperTestSuite)
	}{
		{
			name: "add fee token successfully with token price", // token already register on price contract
			malleate: func() {
				s.SetupPriceContract()
			},
			msg: &types.MsgAddFeeToken{
				Authority: s.feeKeeper.GetAuthority(),
				Config: types.FeeTokenConfiguration{
					Denom:  "factory/orai1wuvhex9xqs3r539mvc6mtm7n20fcj3qr2m0y9khx6n5vtlngfzes3k0rq9/DYeTA4ZQhEwoJ5imjq1Q3zgwfTgkh4WmdfFHAq3jLrv3",
					PoolId: "",
					Status: types.FeeTokenStatus_FROZEN,
				},
			},
			expErr: false,
			assert: func(s *KeeperTestSuite) {
				config, found := s.feeKeeper.GetTokenConfiguration(s.ctx, "factory/orai1wuvhex9xqs3r539mvc6mtm7n20fcj3qr2m0y9khx6n5vtlngfzes3k0rq9/DYeTA4ZQhEwoJ5imjq1Q3zgwfTgkh4WmdfFHAq3jLrv3")
				s.Require().True(found)
				s.Require().Equal(types.FeeTokenStatus_UPDATED, config.Status)

				exchangeRate, found := s.feeKeeper.GetTokenExchangeRate(s.ctx, "factory/orai1wuvhex9xqs3r539mvc6mtm7n20fcj3qr2m0y9khx6n5vtlngfzes3k0rq9/DYeTA4ZQhEwoJ5imjq1Q3zgwfTgkh4WmdfFHAq3jLrv3")
				s.Require().True(found)
				s.Require().NotEmpty(exchangeRate)
			},
		},
		{
			name: "add fee token successfully without token price", // token not register on price contract
			malleate: func() {
				s.SetupPriceContract()
			},
			msg: &types.MsgAddFeeToken{
				Authority: s.feeKeeper.GetAuthority(),
				Config: types.FeeTokenConfiguration{
					Denom:  "denom",
					PoolId: "",
					Status: types.FeeTokenStatus_FROZEN,
				},
			},
			expErr: false,
			assert: func(s *KeeperTestSuite) {
				config, found := s.feeKeeper.GetTokenConfiguration(s.ctx, "denom")
				s.Require().True(found)
				s.Require().Equal(types.FeeTokenStatus_FROZEN, config.Status)

				exchangeRate, found := s.feeKeeper.GetTokenExchangeRate(s.ctx, "denom")
				s.Require().False(found)
				s.Require().Empty(exchangeRate)
			},
		},
	}

	for _, tc := range testCases {
		s.SetupTest()
		tc := tc
		s.Run(tc.name, func() {
			tc.malleate()
			_, err := s.msgServer.AddFeeToken(s.ctx, tc.msg)

			if tc.expErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}

			tc.assert(s)
		})
	}
}
