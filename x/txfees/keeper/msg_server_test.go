package keeper_test

import (
	v05014 "github.com/CosmWasm/wasmd/app/upgrades/v05014"
	"github.com/CosmWasm/wasmd/x/txfees/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
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
				s.Require().NoError(s.feeKeeper.AddBlacklist(s.ctx, target))
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
		{
			// Blacklisting a module account would make the bank send restriction
			// fail every BeginBlocker transfer out of it and halt the chain.
			name:     "add blacklist error module account",
			malleate: func() {},
			msg: &types.MsgAddBlacklist{
				Authority: types.FoundationGroupPolicyAddress,
				Address:   authtypes.NewModuleAddress(minttypes.ModuleName).String(),
			},
			expErr: true,
			assert: func() {
				ok, err := s.feeKeeper.IsBlacklisted(s.ctx, authtypes.NewModuleAddress(minttypes.ModuleName))
				s.Require().NoError(err)
				s.Require().False(ok)
			},
		},
		{
			// Blacklisting the authority itself would freeze the only account
			// able to undo the entry.
			name:     "add blacklist error foundation policy account",
			malleate: func() {},
			msg: &types.MsgAddBlacklist{
				Authority: types.FoundationGroupPolicyAddress,
				Address:   types.FoundationGroupPolicyAddress,
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
				s.Require().NoError(s.feeKeeper.AddBlacklist(s.ctx, target))
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
				s.Require().NoError(s.feeKeeper.AddBlacklist(s.ctx, from))
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
				s.Require().NoError(s.feeKeeper.AddBlacklist(s.ctx, from))
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
		{
			// Recovering into a blacklisted destination would re-freeze the funds.
			name: "error destination is blacklisted",
			malleate: func() {
				s.Require().NoError(s.feeKeeper.AddBlacklist(s.ctx, from))
				s.Require().NoError(s.feeKeeper.AddBlacklist(s.ctx, to))
				s.FundAccount(from, fund)
			},
			msg: &types.MsgRemoveBlacklistAndRecover{
				Authority: types.FoundationGroupPolicyAddress,
				Address:   from.String(),
				ToAddress: to.String(),
			},
			expErr: true,
			assert: func(res *types.MsgRemoveBlacklistAndRecoverResponse) {
				// from must stay blacklisted and keep its balance
				ok, err := s.feeKeeper.IsBlacklisted(s.ctx, from)
				s.Require().NoError(err)
				s.Require().True(ok)
				s.Require().True(s.app.BankKeeper.GetAllBalances(s.ctx, from).Equal(fund))
			},
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

// TestMsgRemoveBlacklistAndRecoverAboveForkHeight exercises the one invariant the
// table-driven test above cannot: RegisterBankSendRestrictions is a no-op at or
// below v05014.ForkHeight, and the suite's default context sits at height 1. Run
// above the fork height so the send restriction is live and the "remove blacklist
// before sweeping" ordering in RemoveBlacklistAndRecover is actually required.
func (s *KeeperTestSuite) TestMsgRemoveBlacklistAndRecoverAboveForkHeight() {
	_, _, from := KeyTestPubAddr()
	_, _, to := KeyTestPubAddr()
	fund := sdk.NewCoins(sdk.NewInt64Coin("orai", 1_000_000))

	s.ctx = s.ctx.WithBlockHeight(v05014.ForkHeight + 1)

	s.FundAccount(from, fund)
	s.Require().NoError(s.feeKeeper.AddBlacklist(s.ctx, from))

	// Sanity: while blacklisted, the bank send restriction blocks outbound sends.
	// Run on a cache context the way baseapp runs a msg: this SDK fork applies
	// subUnlockedCoins before sendRestriction.apply, so the rejected send leaves
	// the sender debited in the context it ran on and only the discard restores it.
	cacheCtx, _ := s.ctx.CacheContext()
	err := s.app.BankKeeper.SendCoins(cacheCtx, from, to, fund)
	s.Require().Error(err)
	s.Require().True(s.app.BankKeeper.GetAllBalances(s.ctx, from).Equal(fund))

	res, err := s.msgServer.RemoveBlacklistAndRecover(s.ctx, &types.MsgRemoveBlacklistAndRecover{
		Authority: types.FoundationGroupPolicyAddress,
		Address:   from.String(),
		ToAddress: to.String(),
	})
	s.Require().NoError(err)
	s.Require().True(res.Amounts.Equal(fund))
	s.Require().True(s.app.BankKeeper.GetAllBalances(s.ctx, from).IsZero())
	s.Require().True(s.app.BankKeeper.GetAllBalances(s.ctx, to).Equal(fund))

	ok, err := s.feeKeeper.IsBlacklisted(s.ctx, from)
	s.Require().NoError(err)
	s.Require().False(ok)
}

// TestMsgAddBlacklistRejectsEveryModuleAccount pins the chain-halt guard across the
// whole module-account set, including modules whose account has not been
// instantiated in the auth store yet (GetAccount returns nil for those, so the
// registered module permissions are the only thing that catches them).
func (s *KeeperTestSuite) TestMsgAddBlacklistRejectsEveryModuleAccount() {
	perms := s.app.AccountKeeper.GetModulePermissions()
	s.Require().NotEmpty(perms)

	for name, perm := range perms {
		addr := perm.GetAddress()

		_, err := s.msgServer.AddBlacklist(s.ctx, &types.MsgAddBlacklist{
			Authority: types.FoundationGroupPolicyAddress,
			Address:   addr.String(),
		})
		s.Require().ErrorIs(err, types.ErrUnblacklistableAddress, "module %s", name)

		ok, err := s.feeKeeper.IsBlacklisted(s.ctx, addr)
		s.Require().NoError(err)
		s.Require().False(ok, "module %s", name)
	}
}

// TestMsgAddBlacklistAllowsIBCEscrowAddress pins the other side of the
// module-account guard: an IBC transfer escrow address MUST stay blacklistable,
// because freezing a channel's escrow is one of the responses to an active
// exploit.
//
// An escrow address is not a module account: it is the ADR-028 hash of
// "<version>\x00<port>/<channel>" (ibctransfertypes.GetEscrowAddress), it is
// absent from the auth module permissions, and the bank module instantiates it
// as a plain BaseAccount the first time coins land on it. Both arms of
// assertBlacklistable must therefore miss it.
func (s *KeeperTestSuite) TestMsgAddBlacklistAllowsIBCEscrowAddress() {
	escrow := ibctransfertypes.GetEscrowAddress(ibctransfertypes.PortID, "channel-0")

	// Blacklistable before the account exists in the auth store...
	_, err := s.msgServer.AddBlacklist(s.ctx, &types.MsgAddBlacklist{
		Authority: types.FoundationGroupPolicyAddress,
		Address:   escrow.String(),
	})
	s.Require().NoError(err)

	ok, err := s.feeKeeper.IsBlacklisted(s.ctx, escrow)
	s.Require().NoError(err)
	s.Require().True(ok)

	// ...and still blacklistable once bank has instantiated it as a BaseAccount.
	s.Require().NoError(s.feeKeeper.RemoveBlacklist(s.ctx, escrow))
	s.FundAccount(escrow, sdk.NewCoins(sdk.NewInt64Coin("orai", 1_000_000)))

	acc := s.app.AccountKeeper.GetAccount(s.ctx, escrow)
	s.Require().NotNil(acc)
	s.Require().NotImplements((*sdk.ModuleAccountI)(nil), acc)

	_, err = s.msgServer.AddBlacklist(s.ctx, &types.MsgAddBlacklist{
		Authority: types.FoundationGroupPolicyAddress,
		Address:   escrow.String(),
	})
	s.Require().NoError(err)

	ok, err = s.feeKeeper.IsBlacklisted(s.ctx, escrow)
	s.Require().NoError(err)
	s.Require().True(ok)
}

// TestBlacklistMsgsRejectEveryNonFoundationAuthority is the authority guard for all
// three blacklist msgs. Every impostor must be rejected and leave state untouched,
// and the positive control at the end proves the assertions are not vacuous.
//
// Together with the signer plumbing this pins "only FoundationGroupPolicyAddress
// can action the blacklist msgs":
//
//   - All three msgs declare `option (cosmos.msg.v1.signer) = "authority"`, so a tx
//     is only valid if msg.Authority itself signed it.
//   - The policy account is an authtypes.ModuleCredential account, whose
//     VerifySignature always returns false, so it can never sign a tx directly.
//   - x/group doExecuteMsgs -> ensureMsgAuthZ requires every msg signer to equal the
//     policy account, so a proposal cannot smuggle in a different authority.
//   - x/gov SubmitProposal asserts the gov module account is the only signer of all
//     proposal msgs, so gov cannot carry these msgs at all.
//   - x/wasm handler_plugin requires the emitting contract to be the msg signer, so
//     no CosmWasm contract can impersonate the policy.
func (s *KeeperTestSuite) TestBlacklistMsgsRejectEveryNonFoundationAuthority() {
	fund := sdk.NewCoins(sdk.NewInt64Coin("orai", 1_000_000))

	impostors := []struct {
		name      string
		authority string
	}{
		{"x/gov module authority", s.feeKeeper.GetAuthority()},
		{"mint module account", authtypes.NewModuleAddress(minttypes.ModuleName).String()},
		{"ibc transfer module account", authtypes.NewModuleAddress(ibctransfertypes.ModuleName).String()},
		{"empty authority", ""},
		{"malformed bech32", "not-an-address"},
		{"right bytes, wrong prefix", "cosmos1afk9zr2hn2jsac63h4hm60vl9z3e5u69gndzf7c99cqge3vzwjzsqx3ckr"},
	}

	for _, tc := range impostors {
		s.SetupTest()
		tc := tc

		s.Run(tc.name, func() {
			_, _, blacklisted := KeyTestPubAddr()
			_, _, fresh := KeyTestPubAddr()

			s.Require().NoError(s.feeKeeper.AddBlacklist(s.ctx, blacklisted))
			s.FundAccount(blacklisted, fund)

			// AddBlacklist must not add anything.
			_, err := s.msgServer.AddBlacklist(s.ctx, &types.MsgAddBlacklist{
				Authority: tc.authority,
				Address:   fresh.String(),
			})
			s.Require().Error(err)
			ok, qErr := s.feeKeeper.IsBlacklisted(s.ctx, fresh)
			s.Require().NoError(qErr)
			s.Require().False(ok)

			// RemoveBlacklist must not remove anything.
			_, err = s.msgServer.RemoveBlacklist(s.ctx, &types.MsgRemoveBlacklist{
				Authority: tc.authority,
				Address:   blacklisted.String(),
			})
			s.Require().Error(err)
			ok, qErr = s.feeKeeper.IsBlacklisted(s.ctx, blacklisted)
			s.Require().NoError(qErr)
			s.Require().True(ok)

			// RemoveBlacklistAndRecover must move no funds.
			_, err = s.msgServer.RemoveBlacklistAndRecover(s.ctx, &types.MsgRemoveBlacklistAndRecover{
				Authority: tc.authority,
				Address:   blacklisted.String(),
				ToAddress: fresh.String(),
			})
			s.Require().Error(err)
			ok, qErr = s.feeKeeper.IsBlacklisted(s.ctx, blacklisted)
			s.Require().NoError(qErr)
			s.Require().True(ok)
			s.Require().True(s.app.BankKeeper.GetAllBalances(s.ctx, blacklisted).Equal(fund))
			s.Require().True(s.app.BankKeeper.GetAllBalances(s.ctx, fresh).IsZero())
		})
	}

	// Positive control: the real authority does succeed on the same setup, so the
	// rejections above are attributable to the authority check and nothing else.
	s.SetupTest()
	s.Run("foundation policy authority succeeds", func() {
		_, _, blacklisted := KeyTestPubAddr()
		_, _, fresh := KeyTestPubAddr()

		s.Require().NoError(s.feeKeeper.AddBlacklist(s.ctx, blacklisted))
		s.FundAccount(blacklisted, fund)

		_, err := s.msgServer.AddBlacklist(s.ctx, &types.MsgAddBlacklist{
			Authority: types.FoundationGroupPolicyAddress,
			Address:   fresh.String(),
		})
		s.Require().NoError(err)
		s.Require().NoError(s.feeKeeper.RemoveBlacklist(s.ctx, fresh))

		res, err := s.msgServer.RemoveBlacklistAndRecover(s.ctx, &types.MsgRemoveBlacklistAndRecover{
			Authority: types.FoundationGroupPolicyAddress,
			Address:   blacklisted.String(),
			ToAddress: fresh.String(),
		})
		s.Require().NoError(err)
		s.Require().True(res.Amounts.Equal(fund))
	})
}

// eventsOfType returns the attributes of every event of the given type, as a
// map per event.
func eventsOfType(events sdk.Events, eventType string) []map[string]string {
	var out []map[string]string
	for _, ev := range events {
		if ev.Type != eventType {
			continue
		}
		attrs := make(map[string]string, len(ev.Attributes))
		for _, attr := range ev.Attributes {
			attrs[attr.Key] = attr.Value
		}
		out = append(out, attrs)
	}
	return out
}

// TestBlacklistMsgsEmitEvents pins the audit trail: every blacklist state
// transition must be reconstructable from block results alone, without
// replaying state or diffing the store.
func (s *KeeperTestSuite) TestBlacklistMsgsEmitEvents() {
	_, _, addr := KeyTestPubAddr()

	// --- add ---
	addCtx := s.ctx.WithEventManager(sdk.NewEventManager())
	_, err := s.msgServer.AddBlacklist(addCtx, &types.MsgAddBlacklist{
		Authority: types.FoundationGroupPolicyAddress,
		Address:   addr.String(),
	})
	s.Require().NoError(err)

	added := eventsOfType(addCtx.EventManager().Events(), types.EventTypeAddBlacklist)
	s.Require().Len(added, 1)
	s.Require().Equal(addr.String(), added[0][types.AttributeKeyAddress])
	s.Require().Equal(types.FoundationGroupPolicyAddress, added[0][types.AttributeKeyAuthority])

	// --- remove ---
	rmCtx := s.ctx.WithEventManager(sdk.NewEventManager())
	_, err = s.msgServer.RemoveBlacklist(rmCtx, &types.MsgRemoveBlacklist{
		Authority: types.FoundationGroupPolicyAddress,
		Address:   addr.String(),
	})
	s.Require().NoError(err)

	removed := eventsOfType(rmCtx.EventManager().Events(), types.EventTypeRemoveBlacklist)
	s.Require().Len(removed, 1)
	s.Require().Equal(addr.String(), removed[0][types.AttributeKeyAddress])
	s.Require().Equal(types.FoundationGroupPolicyAddress, removed[0][types.AttributeKeyAuthority])

	// --- recover ---
	_, _, to := KeyTestPubAddr()
	fund := sdk.NewCoins(sdk.NewInt64Coin("orai", 1_000_000))
	s.FundAccount(addr, fund)
	s.Require().NoError(s.feeKeeper.AddBlacklist(s.ctx, addr))

	recCtx := s.ctx.WithEventManager(sdk.NewEventManager())
	_, err = s.msgServer.RemoveBlacklistAndRecover(recCtx, &types.MsgRemoveBlacklistAndRecover{
		Authority: types.FoundationGroupPolicyAddress,
		Address:   addr.String(),
		ToAddress: to.String(),
	})
	s.Require().NoError(err)

	recovered := eventsOfType(recCtx.EventManager().Events(), types.EventTypeRecoverBlacklist)
	s.Require().Len(recovered, 1)
	s.Require().Equal(addr.String(), recovered[0][types.AttributeKeyAddress])
	s.Require().Equal(to.String(), recovered[0][types.AttributeKeyToAddress])
	s.Require().Equal(fund.String(), recovered[0][types.AttributeKeyAmounts])
	s.Require().Equal(types.FoundationGroupPolicyAddress, recovered[0][types.AttributeKeyAuthority])

	// Recover also unfreezes, so a monitor watching only remove_blacklist must
	// still see this account leave the blacklist.
	alsoRemoved := eventsOfType(recCtx.EventManager().Events(), types.EventTypeRemoveBlacklist)
	s.Require().Len(alsoRemoved, 1)
	s.Require().Equal(addr.String(), alsoRemoved[0][types.AttributeKeyAddress])
}

// TestBlacklistMsgsEmitNoEventsOnFailure pins that a rejected msg leaves no
// audit trail behind, so events cannot be mistaken for applied state.
func (s *KeeperTestSuite) TestBlacklistMsgsEmitNoEventsOnFailure() {
	_, _, addr := KeyTestPubAddr()
	_, _, impostor := KeyTestPubAddr()

	failCtx := s.ctx.WithEventManager(sdk.NewEventManager())
	_, err := s.msgServer.AddBlacklist(failCtx, &types.MsgAddBlacklist{
		Authority: impostor.String(),
		Address:   addr.String(),
	})
	s.Require().Error(err)
	s.Require().Empty(eventsOfType(failCtx.EventManager().Events(), types.EventTypeAddBlacklist))

	// Removing an address that is not blacklisted must not emit either.
	failCtx = s.ctx.WithEventManager(sdk.NewEventManager())
	_, err = s.msgServer.RemoveBlacklist(failCtx, &types.MsgRemoveBlacklist{
		Authority: types.FoundationGroupPolicyAddress,
		Address:   addr.String(),
	})
	s.Require().Error(err)
	s.Require().Empty(eventsOfType(failCtx.EventManager().Events(), types.EventTypeRemoveBlacklist))
}
