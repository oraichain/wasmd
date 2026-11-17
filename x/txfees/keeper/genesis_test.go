package keeper_test

import (
	"github.com/CosmWasm/wasmd/x/txfees/keeper"
	"github.com/CosmWasm/wasmd/x/txfees/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TestExportGenesisRoundTripsBlacklist is the regression guard for the export
// gap: ExportGenesis used to emit only Params, so `simd export` followed by a
// restart silently unfroze every blacklisted account.
func (s *KeeperTestSuite) TestExportGenesisRoundTripsBlacklist() {
	_, _, addr1 := KeyTestPubAddr()
	_, _, addr2 := KeyTestPubAddr()

	s.Require().NoError(s.feeKeeper.AddBlacklist(s.ctx, addr1))
	s.Require().NoError(s.feeKeeper.AddBlacklist(s.ctx, addr2))

	exported := s.feeKeeper.ExportGenesis(s.ctx)
	s.Require().ElementsMatch(
		[]string{addr1.String(), addr2.String()},
		exported.BlacklistAddresses,
	)
	s.Require().NoError(exported.Validate())

	// Wipe the store the way a fresh chain starts, then re-import.
	s.Require().NoError(s.feeKeeper.RemoveBlacklist(s.ctx, addr1))
	s.Require().NoError(s.feeKeeper.RemoveBlacklist(s.ctx, addr2))
	for _, addr := range []sdk.AccAddress{addr1, addr2} {
		ok, err := s.feeKeeper.IsBlacklisted(s.ctx, addr)
		s.Require().NoError(err)
		s.Require().False(ok)
	}

	s.feeKeeper.InitGenesis(s.ctx, *exported)

	for _, addr := range []sdk.AccAddress{addr1, addr2} {
		ok, err := s.feeKeeper.IsBlacklisted(s.ctx, addr)
		s.Require().NoError(err)
		s.Require().True(ok, "%s lost across genesis round trip", addr)
	}

	// Exporting the re-imported state must reproduce the same export.
	s.Require().Equal(exported, s.feeKeeper.ExportGenesis(s.ctx))
}

// TestExportGenesisEmptyBlacklist pins that an empty blacklist exports as nil
// rather than an empty-string entry, so DefaultGenesis stays round-trippable.
func (s *KeeperTestSuite) TestExportGenesisEmptyBlacklist() {
	exported := s.feeKeeper.ExportGenesis(s.ctx)
	s.Require().Empty(exported.BlacklistAddresses)
	s.Require().NoError(exported.Validate())

	s.feeKeeper.InitGenesis(s.ctx, *exported)
	s.Require().Empty(s.feeKeeper.ExportGenesis(s.ctx).BlacklistAddresses)
}

func (s *KeeperTestSuite) TestGenesisStateValidateBlacklist() {
	_, _, addr := KeyTestPubAddr()

	testCases := []struct {
		name      string
		addresses []string
		expErr    string
	}{
		{"empty", nil, ""},
		{"valid", []string{addr.String()}, ""},
		{"malformed bech32", []string{"not-an-address"}, "invalid blacklist address"},
		{"wrong prefix", []string{"cosmos1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu"}, "invalid blacklist address"},
		{"duplicate", []string{addr.String(), addr.String()}, "duplicate blacklist address"},
		{
			"foundation policy",
			[]string{types.FoundationGroupPolicyAddress},
			"foundation x/group policy account",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			gs := types.GenesisState{
				Params:             types.DefaultParams(),
				BlacklistAddresses: tc.addresses,
			}
			err := gs.Validate()
			if tc.expErr == "" {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tc.expErr)
			}
		})
	}
}

// TestNewKeeperRejectsNilAccountKeeper pins the invariant that replaced the dead
// `k.accKeeper == nil` branch in assertBlacklistable: a nil AccountKeeper used to
// disable the module-account guard silently, which would make fee_collector
// blacklistable and halt the chain. It is now a wiring-time panic.
func (s *KeeperTestSuite) TestNewKeeperRejectsNilAccountKeeper() {
	s.Require().PanicsWithValue("txfees keeper: nil AccountKeeper", func() {
		keeper.NewKeeper(nil, nil, nil, nil, nil, "")
	})
}
