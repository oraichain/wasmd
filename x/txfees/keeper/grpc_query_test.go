package keeper_test

import (
	"slices"

	"github.com/CosmWasm/wasmd/x/txfees/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestQueryAllowedTokens() {
	testCases := []struct {
		name     string
		malleate func()
		msg      *types.QueryAllowedTokensRequest
		expErr   bool
		assert   func(allowedTokens []string)
	}{
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
