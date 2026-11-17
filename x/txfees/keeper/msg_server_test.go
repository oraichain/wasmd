package keeper_test

import (
	"github.com/CosmWasm/wasmd/x/txfees/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

func (s *KeeperTestSuite) TestMsgAddBlacklist() {
	_, _, target := KeyTestPubAddr()

	testCases := []struct {
		name     string
		malleate func()
		msg      *types.MsgAddBlacklist
		expErr   bool
		assert   func()
	}{
		{
			name:     "add blacklist successfully",
			malleate: func() {},
			msg: &types.MsgAddBlacklist{
				Authority: types.FoundationGroupPolicyAddress,
				Address:   target.String(),
			},
			expErr: false,
			assert: func() {
				ok, err := s.feeKeeper.IsBlacklisted(s.ctx, target)
				s.Require().NoError(err)
				s.Require().True(ok)
			},
		},
		{
			name:     "add blacklist error invalid group policy authority",
			malleate: func() {},
			msg: &types.MsgAddBlacklist{
				Authority: s.feeKeeper.GetAuthority(),
				Address:   target.String(),
			},
			expErr: true,
			assert: func() {},
		},
		{
			name: "add blacklist error already blacklisted",
			malleate: func() {
				s.feeKeeper.AddBlacklist(s.ctx, target)
			},
			msg: &types.MsgAddBlacklist{
				Authority: types.FoundationGroupPolicyAddress,
				Address:   target.String(),
			},
			expErr: true,
			assert: func() {},
		},
		{
			name:     "add blacklist error invalid address",
			malleate: func() {},
			msg: &types.MsgAddBlacklist{
				Authority: types.FoundationGroupPolicyAddress,
				Address:   "not-an-address",
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
			_, err := s.msgServer.AddBlacklist(s.ctx, tc.msg)
			if tc.expErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
			tc.assert()
		})
	}
}

func (s *KeeperTestSuite) TestMsgRemoveBlacklist() {
	_, _, target := KeyTestPubAddr()

	testCases := []struct {
		name     string
		malleate func()
		msg      *types.MsgRemoveBlacklist
		expErr   bool
		assert   func()
	}{
		{
			name: "remove blacklist successfully",
			malleate: func() {
				s.feeKeeper.AddBlacklist(s.ctx, target)
			},
			msg: &types.MsgRemoveBlacklist{
				Authority: types.FoundationGroupPolicyAddress,
				Address:   target.String(),
			},
			expErr: false,
			assert: func() {
				ok, err := s.feeKeeper.IsBlacklisted(s.ctx, target)
				s.Require().NoError(err)
				s.Require().False(ok)
			},
		},
		{
			name:     "remove blacklist error invalid group policy authority",
			malleate: func() {},
			msg: &types.MsgRemoveBlacklist{
				Authority: s.feeKeeper.GetAuthority(),
				Address:   target.String(),
			},
			expErr: true,
			assert: func() {},
		},
		{
			name:     "remove blacklist error not blacklisted",
			malleate: func() {},
			msg: &types.MsgRemoveBlacklist{
				Authority: types.FoundationGroupPolicyAddress,
				Address:   target.String(),
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
			_, err := s.msgServer.RemoveBlacklist(s.ctx, tc.msg)
			if tc.expErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
			tc.assert()
		})
	}
}

func (s *KeeperTestSuite) TestMsgRemoveBlacklistAndRecover() {
	_, _, from := KeyTestPubAddr()
	_, _, to := KeyTestPubAddr()
	fund := sdk.NewCoins(sdk.NewInt64Coin("orai", 1_000_000))

	testCases := []struct {
		name     string
		malleate func()
		msg      *types.MsgRemoveBlacklistAndRecover
		expErr   bool
		assert   func(res *types.MsgRemoveBlacklistAndRecoverResponse)
	}{
		{
			name: "remove blacklist and recover balances successfully",
			malleate: func() {
				s.feeKeeper.AddBlacklist(s.ctx, from)
				s.FundAccount(from, fund)
			},
			msg: &types.MsgRemoveBlacklistAndRecover{
				Authority: types.FoundationGroupPolicyAddress,
				Address:   from.String(),
				ToAddress: to.String(),
			},
			expErr: false,
			assert: func(res *types.MsgRemoveBlacklistAndRecoverResponse) {
				ok, err := s.feeKeeper.IsBlacklisted(s.ctx, from)
				s.Require().NoError(err)
				s.Require().False(ok)
				s.Require().True(res.Amounts.Equal(fund))
				s.Require().True(s.app.BankKeeper.GetAllBalances(s.ctx, from).IsZero())
				s.Require().True(s.app.BankKeeper.GetAllBalances(s.ctx, to).Equal(fund))
			},
		},
		{
			name: "remove blacklist with zero balance succeeds",
			malleate: func() {
				s.feeKeeper.AddBlacklist(s.ctx, from)
			},
			msg: &types.MsgRemoveBlacklistAndRecover{
				Authority: types.FoundationGroupPolicyAddress,
				Address:   from.String(),
				ToAddress: to.String(),
			},
			expErr: false,
			assert: func(res *types.MsgRemoveBlacklistAndRecoverResponse) {
				ok, err := s.feeKeeper.IsBlacklisted(s.ctx, from)
				s.Require().NoError(err)
				s.Require().False(ok)
				s.Require().True(res.Amounts.IsZero())
			},
		},
		{
			name:     "error invalid group policy authority",
			malleate: func() {},
			msg: &types.MsgRemoveBlacklistAndRecover{
				Authority: s.feeKeeper.GetAuthority(),
				Address:   from.String(),
				ToAddress: to.String(),
			},
			expErr: true,
			assert: func(res *types.MsgRemoveBlacklistAndRecoverResponse) {},
		},
		{
			name:     "error not blacklisted",
			malleate: func() {},
			msg: &types.MsgRemoveBlacklistAndRecover{
				Authority: types.FoundationGroupPolicyAddress,
				Address:   from.String(),
				ToAddress: to.String(),
			},
			expErr: true,
			assert: func(res *types.MsgRemoveBlacklistAndRecoverResponse) {},
		},
		{
			name:     "error same from and to address",
			malleate: func() {},
			msg: &types.MsgRemoveBlacklistAndRecover{
				Authority: types.FoundationGroupPolicyAddress,
				Address:   from.String(),
				ToAddress: from.String(),
			},
			expErr: true,
			assert: func(res *types.MsgRemoveBlacklistAndRecoverResponse) {},
		},
	}

	for _, tc := range testCases {
		s.SetupTest()
		tc := tc
		s.Run(tc.name, func() {
			tc.malleate()
			res, err := s.msgServer.RemoveBlacklistAndRecover(s.ctx, tc.msg)
			if tc.expErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
			tc.assert(res)
		})
	}
}
