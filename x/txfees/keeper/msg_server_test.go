package keeper_test

import (
	"github.com/CosmWasm/wasmd/x/txfees/types"
)

func (s *KeeperTestSuite) TestUpdateParams() {
	testCases := []struct {
		name   string
		msg    *types.MsgUpdateParams
		expErr bool
	}{
		{
			name: "update params successfully",
			msg: &types.MsgUpdateParams{
				Authority: s.feeKeeper.GetAuthority(),
				Params: types.Params{
					TokenBaseDenom:       "denom",
					PriceContractAddress: "contract_address",
				},
			},
			expErr: false,
		},
		{
			name: "update params error invalid authority",
			msg: &types.MsgUpdateParams{
				Authority: "invalid_authority",
				Params: types.Params{
					TokenBaseDenom:       "denom",
					PriceContractAddress: "contract_address",
				},
			},
			expErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			_, err := s.msgServer.UpdateParams(s.ctx, tc.msg)
			if tc.expErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestMsgAddFeeToken() {
	testCases := []struct {
		name     string
		malleate func()
		msg      *types.MsgAddFeeToken
		expErr   bool
		assert   func()
	}{
		{
			name: "add fee token successfully with token price", // token already register on price contract
			malleate: func() {
				s.SetupPriceContract()
			},
			msg: &types.MsgAddFeeToken{
				Authority: s.feeKeeper.GetAuthority(),
				Denom:     "factory/orai1wuvhex9xqs3r539mvc6mtm7n20fcj3qr2m0y9khx6n5vtlngfzes3k0rq9/DYeTA4ZQhEwoJ5imjq1Q3zgwfTgkh4WmdfFHAq3jLrv3",
			},
			expErr: false,
			assert: func() {},
		},
		{
			name: "add fee token successfully without token price", // token not register on price contract
			malleate: func() {
				s.SetupPriceContract()
			},
			msg: &types.MsgAddFeeToken{
				Authority: s.feeKeeper.GetAuthority(),
				Denom:     "factory/orai1wuvhex9xqs3r539mvc6mtm7n20fcj3qr2m0y9khx6n5vtlngfzes3k0rq9/DYeTA4ZQhEwoJ5imjq1Q3zgwfTgkh4WmdfFHAq3jLrv3",
			},
			expErr: false,
			assert: func() {},
		},
		{
			name: "add fee token error token already added",
			malleate: func() {
				s.feeKeeper.AddAllowedToken(s.ctx, "denom")
			},
			msg: &types.MsgAddFeeToken{
				Authority: s.feeKeeper.GetAuthority(),
				Denom:     "denom",
			},
			expErr: true,
			assert: func() {},
		},
		{
			name:     "add fee token error invalid authority address",
			malleate: func() {},
			msg: &types.MsgAddFeeToken{
				Authority: "invalid-address",
				Denom:     "factory/orai1wuvhex9xqs3r539mvc6mtm7n20fcj3qr2m0y9khx6n5vtlngfzes3k0rq9/DYeTA4ZQhEwoJ5imjq1Q3zgwfTgkh4WmdfFHAq3jLrv3",
			},
			expErr: true,
			assert: func() {},
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

			tc.assert()
		})
	}
}

func (s *KeeperTestSuite) TestMsgRemoveFeeToken() {
	testCases := []struct {
		name     string
		malleate func()
		msg      *types.MsgRemoveFeeToken
		expErr   bool
		assert   func()
	}{
		{
			name: "remove fee token successfully",
			malleate: func() {
				s.SetupPriceContract()
				msg := &types.MsgAddFeeToken{
					Authority: s.feeKeeper.GetAuthority(),
					Denom:     "factory/orai1wuvhex9xqs3r539mvc6mtm7n20fcj3qr2m0y9khx6n5vtlngfzes3k0rq9/DYeTA4ZQhEwoJ5imjq1Q3zgwfTgkh4WmdfFHAq3jLrv3",
				}
				_, err := s.msgServer.AddFeeToken(s.ctx, msg)
				s.Require().NoError(err)
			},
			msg: &types.MsgRemoveFeeToken{
				Authority: s.feeKeeper.GetAuthority(),
				Denom:     "factory/orai1wuvhex9xqs3r539mvc6mtm7n20fcj3qr2m0y9khx6n5vtlngfzes3k0rq9/DYeTA4ZQhEwoJ5imjq1Q3zgwfTgkh4WmdfFHAq3jLrv3",
			},
			expErr: false,
			assert: func() {
				isAllowed, err := s.feeKeeper.IsTokenAllowed(s.ctx, "factory/orai1wuvhex9xqs3r539mvc6mtm7n20fcj3qr2m0y9khx6n5vtlngfzes3k0rq9/DYeTA4ZQhEwoJ5imjq1Q3zgwfTgkh4WmdfFHAq3jLrv3")
				s.Require().NoError(err)
				s.Require().False(isAllowed)
			},
		},
		{
			name:     "remove fee token error invalid authority",
			malleate: func() {},
			msg: &types.MsgRemoveFeeToken{
				Authority: "invalid_address",
				Denom:     "factory/orai1wuvhex9xqs3r539mvc6mtm7n20fcj3qr2m0y9khx6n5vtlngfzes3k0rq9/DYeTA4ZQhEwoJ5imjq1Q3zgwfTgkh4WmdfFHAq3jLrv3",
			},
			expErr: true,
			assert: func() {},
		},
		{
			name:     "remove fee token error invalid token not registed",
			malleate: func() {},
			msg: &types.MsgRemoveFeeToken{
				Authority: s.feeKeeper.GetAuthority(),
				Denom:     "factory/orai1wuvhex9xqs3r539mvc6mtm7n20fcj3qr2m0y9khx6n5vtlngfzes3k0rq9/DYeTA4ZQhEwoJ5imjq1Q3zgwfTgkh4WmdfFHAq3jLrv3",
			},
			expErr: true,
			assert: func() {},
		},
	}

	for _, tc := range testCases {
		s.SetupTest()
		tc := tc
		s.Run(tc.name, func() {
			tc.malleate()
			_, err := s.msgServer.RemoveFeeToken(s.ctx, tc.msg)

			if tc.expErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}

			tc.assert()
		})
	}

}
