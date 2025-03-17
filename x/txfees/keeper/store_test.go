package keeper_test

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
