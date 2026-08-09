package v10_test

import (
	"fmt"
	"testing"
	"time"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/suite"

	wasmApp "github.com/CosmWasm/wasmd/app"
	v10 "github.com/CosmWasm/wasmd/app/upgrades/v05014"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type UpgradeTestSuite struct {
	suite.Suite

	App *wasmApp.WasmApp
	Ctx sdk.Context
}

func TestUpgradeTestSuite(t *testing.T) {
	suite.Run(t, new(UpgradeTestSuite))
}

func (s *UpgradeTestSuite) SetupTest() {
	s.App = wasmApp.Setup(s.T())
	s.Ctx = s.App.NewContextLegacy(false, cmtproto.Header{
		Height: 1,
		Time:   time.Now().UTC(),
	})
}

// BeginNewBlock advances one block via app BeginBlocker (which runs fork logic).
// The bool is kept for API compatibility with Osmosis helpers and is unused.
func (s *UpgradeTestSuite) BeginNewBlock(_ bool) {
	newHeight := s.Ctx.BlockHeight() + 1
	newTime := s.Ctx.BlockTime().Add(time.Second)
	header := cmtproto.Header{Height: newHeight, Time: newTime}

	s.Ctx = s.Ctx.WithBlockHeight(newHeight).WithBlockTime(newTime)

	_, err := s.App.BeginBlocker(s.Ctx)
	s.Require().NoError(err)

	s.Ctx = s.App.NewContextLegacy(false, header)
}

func (s *UpgradeTestSuite) TestUpgradePayments() {
	testCases := []struct {
		msg     string
		upgrade func()
	}{
		{
			"Test that upgrade succeeds",
			func() {
				// First run block N-1, BeginNewBlock takes ctx height + 1
				s.Ctx = s.Ctx.WithBlockHeight(v10.ForkHeight - 2)
				s.BeginNewBlock(false)

				// run upgrade height
				s.Require().NotPanics(func() {
					s.BeginNewBlock(false)
				})
			},
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset
			tc.upgrade()
		})
	}
}
