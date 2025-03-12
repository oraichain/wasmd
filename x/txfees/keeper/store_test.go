package keeper_test

import (
	"cosmossdk.io/math"
	"github.com/CosmWasm/wasmd/x/txfees/types"
)

func (s *KeeperTestSuite) TestStoreAllowedToken() {
	allowedTokenList := []string{"token1", "token2"}
	unallowedToken := "token3"

	for _, token := range allowedTokenList {
		s.feeKeeper.AddAllowedToken(s.ctx, token)
		isAllowed, err := s.feeKeeper.IsTokenAllowed(s.ctx, token)
		s.Require().True(isAllowed)
		s.Require().NoError(err)
	}

	isAllowed, err := s.feeKeeper.IsTokenAllowed(s.ctx, unallowedToken)
	s.Require().False(isAllowed)
	s.Require().NoError(err)

	var allowedList []string
	s.feeKeeper.IterateAllowedTokenList(s.ctx, func(denom string) (stop bool) {
		allowedList = append(allowedList, denom)
		return false
	})
	s.Require().Equal(allowedTokenList, allowedList)

	for _, token := range allowedTokenList {
		s.feeKeeper.RemoveAllowedToken(s.ctx, token)
		isAllowed, err := s.feeKeeper.IsTokenAllowed(s.ctx, token)
		s.Require().False(isAllowed)
		s.Require().NoError(err)
	}
}

func (s *KeeperTestSuite) TestStoreTokenConfiguration() {
	config := types.FeeTokenConfiguration{
		Denom:  "denom",
		PoolId: "PoolId",
		Status: types.FeeTokenStatus_UPDATED,
	}

	// set to store
	s.feeKeeper.SetTokenConfiguration(s.ctx, config)

	// get from store
	storedConfig, found := s.feeKeeper.GetTokenConfiguration(s.ctx, "denom")
	s.Require().True(found)
	s.Require().Equal(config, storedConfig)

	s.feeKeeper.RemoveTokenConfiguration(s.ctx, "denom")
	_, found = s.feeKeeper.GetTokenConfiguration(s.ctx, "denom")
	s.Require().False(found)
}

func (s *KeeperTestSuite) TestStoreTokenExchangeRate() {
	rate := math.LegacyOneDec()

	s.feeKeeper.SetTokenExchangeRate(s.ctx, "denom", rate)

	storedRate, found := s.feeKeeper.GetTokenExchangeRate(s.ctx, "denom")
	s.Require().True(found)
	s.Require().Equal(rate, storedRate)

	err := s.feeKeeper.RemoveTokenExchangeRate(s.ctx, "denom")
	s.Require().NoError(err)

	_, found = s.feeKeeper.GetTokenExchangeRate(s.ctx, "denom")
	s.Require().False(found)
}
