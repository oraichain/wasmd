package keeper_test

import (
	"slices"

	"github.com/CosmWasm/wasmd/x/txfees/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
)

func (s *KeeperTestSuite) TestQueryCheckBlacklist() {
	_, _, addr := KeyTestPubAddr()

	res, err := s.queryClient.CheckBlacklist(s.ctx, &types.QueryCheckBlacklistRequest{Address: addr.String()})
	s.Require().NoError(err)
	s.Require().False(res.Blacklisted)

	s.Require().NoError(s.feeKeeper.AddBlacklist(s.ctx, addr))

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

	s.Require().NoError(s.feeKeeper.AddBlacklist(s.ctx, addr1))
	s.Require().NoError(s.feeKeeper.AddBlacklist(s.ctx, addr2))

	res, err = s.queryClient.BlacklistAddresses(s.ctx, &types.QueryBlacklistAddressesRequest{})
	s.Require().NoError(err)
	s.Require().ElementsMatch([]string{addr1.String(), addr2.String()}, res.Addresses)
}

// TestQueryBlacklistAddressesPagination pins the bound on this query: it is
// exposed over gRPC/REST with no gas meter, so it must page instead of
// iterating the whole prefix into one response.
func (s *KeeperTestSuite) TestQueryBlacklistAddressesPagination() {
	var all []string
	for i := 0; i < 5; i++ {
		_, _, addr := KeyTestPubAddr()
		s.Require().NoError(s.feeKeeper.AddBlacklist(s.ctx, addr))
		all = append(all, addr.String())
	}

	var paged []string
	var nextKey []byte
	for page := 0; page < 5; page++ {
		res, err := s.queryClient.BlacklistAddresses(s.ctx, &types.QueryBlacklistAddressesRequest{
			Pagination: &query.PageRequest{Key: nextKey, Limit: 2},
		})
		s.Require().NoError(err)
		s.Require().LessOrEqual(len(res.Addresses), 2)
		paged = append(paged, res.Addresses...)

		s.Require().NotNil(res.Pagination)
		nextKey = res.Pagination.NextKey
		if len(nextKey) == 0 {
			break
		}
	}

	s.Require().Empty(nextKey)
	s.Require().ElementsMatch(all, paged)
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
