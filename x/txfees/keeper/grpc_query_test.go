package keeper_test

import (
	"slices"

	"github.com/CosmWasm/wasmd/x/txfees/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestQueryCheckBlacklist() {
	_, _, addr := KeyTestPubAddr()

	res, err := s.queryClient.CheckBlacklist(s.ctx, &types.QueryCheckBlacklistRequest{Address: addr.String()})
	s.Require().NoError(err)
	s.Require().False(res.Blacklisted)

	s.feeKeeper.AddBlacklist(s.ctx, addr)

	res, err = s.queryClient.CheckBlacklist(s.ctx, &types.QueryCheckBlacklistRequest{Address: addr.String()})
	s.Require().NoError(err)
	s.Require().True(res.Blacklisted)
}

func (s *KeeperTestSuite) TestQueryBlacklistAddresses() {
	_, _, addr1 := KeyTestPubAddr()
	_, _, addr2 := KeyTestPubAddr()

	res, err := s.queryClient.BlacklistAddresses(s.ctx, &types.QueryBlacklistAddressesRequest{})
	s.Require().NoError(err)
	s.Require().Empty(res.Addresses)

	s.feeKeeper.AddBlacklist(s.ctx, addr1)
	s.feeKeeper.AddBlacklist(s.ctx, addr2)

	res, err = s.queryClient.BlacklistAddresses(s.ctx, &types.QueryBlacklistAddressesRequest{})
	s.Require().NoError(err)
	s.Require().ElementsMatch([]string{addr1.String(), addr2.String()}, res.Addresses)
}

func (s *KeeperTestSuite) TestQueryAllowedTokens() {
	testCases := []struct {
		name     string
		malleate func()
		msg      *types.QueryAllowedTokensRequest
		expErr   bool
		assert   func(allowedTokens []string)
	}{
		{
			name:     "query allowed tokens successfully with 0 allowed token",
			malleate: func() {},
			msg:      &types.QueryAllowedTokensRequest{},
			expErr:   false,
			assert: func(allowedTokens []string) {
				expectedDenoms := []string(nil)

				slices.Sort(expectedDenoms)
				slices.Sort(allowedTokens)

				s.Require().Equal(expectedDenoms, allowedTokens)
			},
		},
		{
			name: "query allowed tokens successfully with 1 allowed token",
			malleate: func() {
				sdkCtx := sdk.UnwrapSDKContext(s.ctx)
				denoms := []string{"usdai"}

				for _, denom := range denoms {
					s.feeKeeper.AddAllowedToken(sdkCtx, denom)
				}
			},
			msg:    &types.QueryAllowedTokensRequest{},
			expErr: false,
			assert: func(allowedTokens []string) {
				expectedDenoms := []string{"usdai"}

				slices.Sort(expectedDenoms)
				slices.Sort(allowedTokens)

				s.Require().Equal(expectedDenoms, allowedTokens)
			},
		},
		{
			name: "query allowed tokens successfully with more than 1 allowed token",
			malleate: func() {
				sdkCtx := sdk.UnwrapSDKContext(s.ctx)
				denoms := []string{"usdai", "usdtai", "usdcai"}

				for _, denom := range denoms {
					s.feeKeeper.AddAllowedToken(sdkCtx, denom)
				}
			},
			msg:    &types.QueryAllowedTokensRequest{},
			expErr: false,
			assert: func(allowedTokens []string) {
				expectedDenoms := []string{"usdai", "usdtai", "usdcai"}

				slices.Sort(expectedDenoms)
				slices.Sort(allowedTokens)

				s.Require().Equal(expectedDenoms, allowedTokens)
			},
		},
	}

	for _, tc := range testCases {
		s.SetupTest()
		tc := tc

		s.Run(tc.name, func() {
			tc.malleate()
			res, err := s.queryClient.AllowedTokens(s.ctx, tc.msg)

			if tc.expErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}

			tc.assert(res.Tokens)
		})
	}
}
