package keeper_test

import (
	"time"

	"github.com/CosmWasm/wasmd/x/txfees/types"
)

func (s *KeeperTestSuite) TestStoreEpochInfo() {
	s.SetupTest()
	expected := types.EpochInfo{
		Identifier:              "Testing",
		StartTime:               time.Now().UTC(),
		Duration:                10,
		CurrentEpoch:            0,
		CurrentEpochStartTime:   time.Now().UTC(),
		EpochCountingStarted:    false,
		CurrentEpochStartHeight: 0,
	}
	err := s.feeKeeper.AddEpochInfo(s.ctx, expected)
	s.Require().NoError(err)

	storedEpoch, found := s.feeKeeper.GetEpochInfo(s.ctx, expected.Identifier)
	s.Require().True(found)
	s.Require().Equal(expected.StartTime, storedEpoch.StartTime)
	s.Require().Equal(expected.Duration, storedEpoch.Duration)
	s.Require().Equal(expected.EpochCountingStarted, storedEpoch.EpochCountingStarted)

	found, err = s.feeKeeper.HasEpochInfo(s.ctx, expected.Identifier)
	s.Require().NoError(err)
	s.Require().True(found)
}
