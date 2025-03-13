package keeper_test

import (
	"slices"
	"sort"

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

func (s *KeeperTestSuite) TestQueryTokensConfig() {
	testCases := []struct {
		name     string
		malleate func()
		msg      *types.QueryTokensConfigRequest
		expErr   bool
		assert   func(tokenConfigs []types.FeeTokenConfiguration)
	}{
		{
			name:     "query tokens config successfully with 0 token",
			malleate: func() {},
			msg:      &types.QueryTokensConfigRequest{},
			expErr:   false,
			assert: func(tokenConfigs []types.FeeTokenConfiguration) {
				expectedTokensConfig := []types.FeeTokenConfiguration(nil)

				sort.Slice(expectedTokensConfig, func(i, j int) bool {
					return expectedTokensConfig[i].Denom < expectedTokensConfig[j].Denom
				})

				sort.Slice(tokenConfigs, func(i, j int) bool {
					return tokenConfigs[i].Denom < tokenConfigs[j].Denom
				})

				s.Require().Equal(expectedTokensConfig, tokenConfigs)
			},
		},
		{
			name: "query tokens config successfully with 1 token",
			malleate: func() {
				sdkCtx := sdk.UnwrapSDKContext(s.ctx)
				configs := []types.FeeTokenConfiguration{
					{
						Denom:  "usdai",
						PoolId: "some_id",
						Status: types.FeeTokenStatus_FROZEN,
					},
				}

				for _, config := range configs {
					s.feeKeeper.AddAllowedToken(sdkCtx, config.Denom)
					s.feeKeeper.SetTokenConfiguration(sdkCtx, config)
				}
			},
			msg:    &types.QueryTokensConfigRequest{},
			expErr: false,
			assert: func(tokenConfigs []types.FeeTokenConfiguration) {
				expectedTokensConfig := []types.FeeTokenConfiguration{
					{
						Denom:  "usdai",
						PoolId: "some_id",
						Status: types.FeeTokenStatus_FROZEN,
					},
				}

				sort.Slice(expectedTokensConfig, func(i, j int) bool {
					return expectedTokensConfig[i].Denom < expectedTokensConfig[j].Denom
				})

				sort.Slice(tokenConfigs, func(i, j int) bool {
					return tokenConfigs[i].Denom < tokenConfigs[j].Denom
				})

				s.Require().Equal(expectedTokensConfig, tokenConfigs)
			},
		},
		{
			name: "query tokens config successfully with more than 1 token",
			malleate: func() {
				sdkCtx := sdk.UnwrapSDKContext(s.ctx)
				configs := []types.FeeTokenConfiguration{
					{
						Denom:  "usdai",
						PoolId: "some_id",
						Status: types.FeeTokenStatus_FROZEN,
					},
					{
						Denom:  "usdtai",
						PoolId: "some_id",
						Status: types.FeeTokenStatus_FROZEN,
					},
					{
						Denom:  "usdcai",
						PoolId: "some_id",
						Status: types.FeeTokenStatus_FROZEN,
					},
				}

				for _, config := range configs {
					s.feeKeeper.AddAllowedToken(sdkCtx, config.Denom)
					s.feeKeeper.SetTokenConfiguration(sdkCtx, config)
				}
			},
			msg:    &types.QueryTokensConfigRequest{},
			expErr: false,
			assert: func(tokenConfigs []types.FeeTokenConfiguration) {
				expectedTokensConfig := []types.FeeTokenConfiguration{
					{
						Denom:  "usdai",
						PoolId: "some_id",
						Status: types.FeeTokenStatus_FROZEN,
					},
					{
						Denom:  "usdtai",
						PoolId: "some_id",
						Status: types.FeeTokenStatus_FROZEN,
					},
					{
						Denom:  "usdcai",
						PoolId: "some_id",
						Status: types.FeeTokenStatus_FROZEN,
					},
				}

				sort.Slice(expectedTokensConfig, func(i, j int) bool {
					return expectedTokensConfig[i].Denom < expectedTokensConfig[j].Denom
				})

				sort.Slice(tokenConfigs, func(i, j int) bool {
					return tokenConfigs[i].Denom < tokenConfigs[j].Denom
				})

				s.Require().Equal(expectedTokensConfig, tokenConfigs)
			},
		},
	}

	for _, tc := range testCases {
		s.SetupTest()
		tc := tc

		s.Run(tc.name, func() {
			tc.malleate()
			res, err := s.queryClient.TokensConfig(s.ctx, tc.msg)

			if tc.expErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}

			tc.assert(res.Configs)
		})
	}
}

func (s *KeeperTestSuite) TestQueryTokenConfig() {
	testCases := []struct {
		name     string
		malleate func()
		msg      *types.QueryTokenConfigRequest
		expErr   bool
		assert   func(tokenConfig types.FeeTokenConfiguration)
	}{
		{
			name:     "query token config failed with 0 token",
			malleate: func() {},
			msg:      &types.QueryTokenConfigRequest{Denom: "usdai"},
			expErr:   true,
			assert:   func(tokenConfig types.FeeTokenConfiguration) {},
		},
		{
			name: "query token config successfully with 1 token",
			malleate: func() {
				sdkCtx := sdk.UnwrapSDKContext(s.ctx)
				configs := []types.FeeTokenConfiguration{
					{
						Denom:  "usdai",
						PoolId: "some_id",
						Status: types.FeeTokenStatus_FROZEN,
					},
				}

				for _, config := range configs {
					s.feeKeeper.SetTokenConfiguration(sdkCtx, config)
				}
			},
			msg:    &types.QueryTokenConfigRequest{Denom: "usdai"},
			expErr: false,
			assert: func(tokenConfig types.FeeTokenConfiguration) {
				expectedTokenConfig := types.FeeTokenConfiguration{
					Denom:  "usdai",
					PoolId: "some_id",
					Status: types.FeeTokenStatus_FROZEN,
				}

				s.Require().Equal(expectedTokenConfig, tokenConfig)
			},
		},
	}

	for _, tc := range testCases {
		s.SetupTest()
		tc := tc

		s.Run(tc.name, func() {
			tc.malleate()
			res, err := s.queryClient.TokenConfig(s.ctx, tc.msg)

			if tc.expErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)

				tc.assert(res.Config)
			}

		})
	}
}
