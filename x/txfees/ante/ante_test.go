package ante_test

import (
	"time"

	"cosmossdk.io/math"
	"cosmossdk.io/x/feegrant"
	txfeesante "github.com/CosmWasm/wasmd/x/txfees/ante"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
	authz "github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

var testGasLimit uint64 = 200_000

func (s *AnteTestSuite) TestTxFeesAntehandler() {

	denominator := int64(100000)
	med := math.LegacyNewDec(200).Quo(math.LegacyNewDec(denominator)) // 0.002
	minGasPrices := []sdk.DecCoin{sdk.NewDecCoinFromDec("orai", med)}

	testCases := map[string]struct {
		malleate    func(msg sdk.Msg, feeAmount sdk.Coins, gasLimit uint64) signing.Tx
		minGasPrice []sdk.DecCoin
		feeAmount   sdk.Coins
		gasLimit    uint64
		txMsg       sdk.Msg
		txCheck     bool
		expErr      bool
	}{
		"successfully with orai": {
			malleate: func(msg sdk.Msg, feeAmount sdk.Coins, gasLimit uint64) signing.Tx {
				s.FundAccount(s.TestAccount.acc.GetAddress(), sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(100000))))
				privs, accNums, accSeqs := []cryptotypes.PrivKey{s.TestAccount.priv}, []uint64{s.TestAccount.acc.GetAccountNumber()}, []uint64{s.TestAccount.acc.GetSequence()}
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(gasLimit)
				tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
				s.Require().NoError(err)

				return tx
			},
			minGasPrice: minGasPrices,
			feeAmount:   sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(1000))),
			gasLimit:    testGasLimit,

			txCheck: true,
			expErr:  false,
		},
		"successfully with new fee token": {
			malleate: func(msg sdk.Msg, feeAmount sdk.Coins, gasLimit uint64) signing.Tx {
				// setup price contract
				s.SetupPriceContract()

				// add token
				s.tfk.AddAllowedToken(s.ctx, "usdai")

				s.FundAccount(s.TestAccount.acc.GetAddress(), sdk.NewCoins(sdk.NewCoin("usdai", math.NewInt(100000))))
				privs, accNums, accSeqs := []cryptotypes.PrivKey{s.TestAccount.priv}, []uint64{s.TestAccount.acc.GetAccountNumber()}, []uint64{s.TestAccount.acc.GetSequence()}
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(gasLimit)
				tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
				s.Require().NoError(err)

				return tx
			},
			minGasPrice: minGasPrices,
			feeAmount:   sdk.NewCoins(sdk.NewCoin("usdai", math.NewInt(2000))),
			gasLimit:    testGasLimit,

			txCheck: true,
			expErr:  false,
		},
		"error with new fee token insufficient fees": {
			malleate: func(msg sdk.Msg, feeAmount sdk.Coins, gasLimit uint64) signing.Tx {
				// setup price contract
				s.SetupPriceContract()

				// add token
				s.tfk.AddAllowedToken(s.ctx, "usdai")

				s.FundAccount(s.TestAccount.acc.GetAddress(), sdk.NewCoins(sdk.NewCoin("usdai", math.NewInt(100000))))
				privs, accNums, accSeqs := []cryptotypes.PrivKey{s.TestAccount.priv}, []uint64{s.TestAccount.acc.GetAccountNumber()}, []uint64{s.TestAccount.acc.GetSequence()}
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(gasLimit)
				tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
				s.Require().NoError(err)

				return tx
			},
			minGasPrice: minGasPrices,
			feeAmount:   sdk.NewCoins(sdk.NewCoin("usdai", math.NewInt(200))),
			gasLimit:    testGasLimit,

			txCheck: true,
			expErr:  true,
		},
		"error more than 1 token fees": {
			malleate: func(msg sdk.Msg, feeAmount sdk.Coins, gasLimit uint64) signing.Tx {
				// setup price contract
				s.SetupPriceContract()

				// add token
				s.tfk.AddAllowedToken(s.ctx, "usdai")

				s.FundAccount(s.TestAccount.acc.GetAddress(), sdk.NewCoins(sdk.NewCoin("usdai", math.NewInt(100000))))
				privs, accNums, accSeqs := []cryptotypes.PrivKey{s.TestAccount.priv}, []uint64{s.TestAccount.acc.GetAccountNumber()}, []uint64{s.TestAccount.acc.GetSequence()}
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(gasLimit)
				tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
				s.Require().NoError(err)

				return tx
			},
			minGasPrice: minGasPrices,
			feeAmount:   sdk.NewCoins(sdk.NewCoin("usdai", math.NewInt(200)), sdk.NewCoin("orai", math.NewInt(100))),
			gasLimit:    testGasLimit,

			txCheck: true,
			expErr:  true,
		},
		"error query fees token": {
			malleate: func(msg sdk.Msg, feeAmount sdk.Coins, gasLimit uint64) signing.Tx {
				// setup price contract
				s.SetupPriceContract()

				// add token
				s.tfk.AddAllowedToken(s.ctx, "usdc") // we allow this token but price contract not have usdc data => query error

				s.FundAccount(s.TestAccount.acc.GetAddress(), sdk.NewCoins(sdk.NewCoin("usdc", math.NewInt(100000))))
				privs, accNums, accSeqs := []cryptotypes.PrivKey{s.TestAccount.priv}, []uint64{s.TestAccount.acc.GetAccountNumber()}, []uint64{s.TestAccount.acc.GetSequence()}
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(gasLimit)
				tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
				s.Require().NoError(err)

				return tx
			},
			minGasPrice: minGasPrices,
			feeAmount:   sdk.NewCoins(sdk.NewCoin("usdc", math.NewInt(200))),
			gasLimit:    testGasLimit,

			txCheck: true,
			expErr:  true,
		},
	}

	for name, tc := range testCases {
		s.Run(name, func() {
			s.SetupTest()
			accAddress := s.TestAccount.acc.GetAddress()
			txMsg := testdata.NewTestMsg(accAddress)

			tx := tc.malleate(txMsg, tc.feeAmount, tc.gasLimit)
			s.ctx = s.ctx.WithIsCheckTx(tc.txCheck)
			_, err := s.anteHandler(s.ctx.WithMinGasPrices(tc.minGasPrice), tx, false)
			if !tc.expErr {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *AnteTestSuite) TestDeductFeeAnteHandle() {
	testCases := map[string]struct {
		malleate        func(msg sdk.Msg, feeAmount sdk.Coins, gasLimit uint64, feeGranterAcct *TestAccount) signing.Tx
		feeAmount       sdk.Coins
		gasLimit        uint64
		setupFeeGranter bool
		fundFeeGranter  bool
		expErr          bool
		checkFeeDeduct  bool
	}{
		"successfully deduct fee from fee payer": {
			malleate: func(msg sdk.Msg, feeAmount sdk.Coins, gasLimit uint64, feeGranterAcct *TestAccount) signing.Tx {
				s.FundAccount(s.TestAccount.acc.GetAddress(), sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(100000))))
				privs, accNums, accSeqs := []cryptotypes.PrivKey{s.TestAccount.priv}, []uint64{s.TestAccount.acc.GetAccountNumber()}, []uint64{s.TestAccount.acc.GetSequence()}
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(gasLimit)
				tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
				s.Require().NoError(err)
				return tx
			},
			feeAmount:       sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(1000))),
			gasLimit:        testGasLimit,
			setupFeeGranter: false,
			fundFeeGranter:  false,
			expErr:          false,
			checkFeeDeduct:  true,
		},
		"successfully deduct fee from fee granter": {
			malleate: func(msg sdk.Msg, feeAmount sdk.Coins, gasLimit uint64, feeGranterAcct *TestAccount) signing.Tx {
				// Fund only the fee payer, fee granter is funded in setup
				feeGranter := s.TestAccounts[1]
				s.FundAccount(feeGranter.acc.GetAddress(), sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(100000))))
				// Create a basic fee allowance between the fee granter and the fee payer
				// This is required for fee granting to work
				grantee := s.TestAccount.acc.GetAddress()

				// Create a basic allowance with a spending limit and expiration
				expiration := time.Now().Add(time.Hour)
				basicAllowance := &feegrant.BasicAllowance{
					SpendLimit: sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(100000))),
					Expiration: &expiration,
				}

				err := s.fgk.GrantAllowance(s.ctx, feeGranter.acc.GetAddress(), grantee, basicAllowance)
				s.Require().NoError(err)

				privs, accNums, accSeqs := []cryptotypes.PrivKey{feeGranter.priv}, []uint64{feeGranter.acc.GetAccountNumber()}, []uint64{feeGranter.acc.GetSequence()}
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(gasLimit)
				s.txBuilder.SetFeeGranter(feeGranter.acc.GetAddress())
				tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
				s.Require().NoError(err)
				return tx
			},
			feeAmount:       sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(1000))),
			gasLimit:        testGasLimit,
			setupFeeGranter: true,
			fundFeeGranter:  true,
			expErr:          false,
			checkFeeDeduct:  true,
		},
		"error when fee granter has insufficient funds": {
			malleate: func(msg sdk.Msg, feeAmount sdk.Coins, gasLimit uint64, feeGranterAcct *TestAccount) signing.Tx {
				// Fund only the fee payer, not the fee granter (fee granter should have zero balance)
				s.FundAccount(s.TestAccount.acc.GetAddress(), sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(100000))))

				privs, accNums, accSeqs := []cryptotypes.PrivKey{s.TestAccount.priv}, []uint64{s.TestAccount.acc.GetAccountNumber()}, []uint64{s.TestAccount.acc.GetSequence()}
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(gasLimit)
				s.txBuilder.SetFeeGranter(feeGranterAcct.acc.GetAddress())
				tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
				s.Require().NoError(err)
				return tx
			},
			feeAmount:       sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(1000))),
			gasLimit:        testGasLimit,
			setupFeeGranter: true,
			fundFeeGranter:  false,
			expErr:          true,
			checkFeeDeduct:  false,
		},
		"error when fee payer does not exist": {
			malleate: func(msg sdk.Msg, feeAmount sdk.Coins, gasLimit uint64, feeGranterAcct *TestAccount) signing.Tx {
				// Create a new account that doesn't exist in the store
				nonExistentAcc := sdk.AccAddress("non-existent-acc")
				privs, accNums, accSeqs := []cryptotypes.PrivKey{s.TestAccount.priv}, []uint64{0}, []uint64{0}
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				s.txBuilder.SetFeeAmount(feeAmount)
				s.txBuilder.SetGasLimit(gasLimit)
				s.txBuilder.SetFeePayer(nonExistentAcc)
				tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
				s.Require().NoError(err)
				return tx
			},
			feeAmount:       sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(1000))),
			gasLimit:        testGasLimit,
			setupFeeGranter: false,
			fundFeeGranter:  false,
			expErr:          true,
			checkFeeDeduct:  false,
		},
	}

	for name, tc := range testCases {
		s.Run(name, func() {
			s.SetupTest()
			accAddress := s.TestAccount.acc.GetAddress()
			txMsg := testdata.NewTestMsg(accAddress)

			// Create fee granter account only once at the beginning of the test if needed
			var feeGranterAcct *TestAccount
			if tc.setupFeeGranter {
				feeGranterAcct = &s.TestAccounts[1]
			}

			tx := tc.malleate(txMsg, tc.feeAmount, tc.gasLimit, feeGranterAcct)
			_, err := s.anteHandler(s.ctx, tx, false)
			if !tc.expErr {
				s.Require().NoError(err)
				// Verify fee deduction if needed
				if tc.checkFeeDeduct {
					if tc.setupFeeGranter {
						// Check fee granter's balance was reduced
						balance := s.bk.GetBalance(s.ctx, feeGranterAcct.acc.GetAddress(), "orai")
						s.Require().Equal(math.NewInt(99000), balance.Amount)
					} else {
						// Check fee payer's balance was reduced
						balance := s.bk.GetBalance(s.ctx, accAddress, "orai")
						s.Require().Equal(math.NewInt(99000), balance.Amount)
					}
				}
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *AnteTestSuite) TestBlacklistDecorator() {
	type tc struct {
		name      string
		malleate  func() signing.Tx
		expErr    bool
		errSubstr string
	}

	cases := []tc{
		{
			name: "non-blacklisted signer allowed",
			malleate: func() signing.Tx {
				s.FundAccount(s.TestAccount.acc.GetAddress(), sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(100000))))
				msg := testdata.NewTestMsg(s.TestAccount.acc.GetAddress())
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				s.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(1000))))
				s.txBuilder.SetGasLimit(testGasLimit)
				tx, err := s.CreateTestTx(
					[]cryptotypes.PrivKey{s.TestAccount.priv},
					[]uint64{s.TestAccount.acc.GetAccountNumber()},
					[]uint64{s.TestAccount.acc.GetSequence()},
					s.ctx.ChainID(),
				)
				s.Require().NoError(err)
				return tx
			},
			expErr: false,
		},
		{
			name: "blacklisted signer rejected",
			malleate: func() signing.Tx {
				s.tfk.AddBlacklist(s.ctx, s.TestAccount.acc.GetAddress())
				s.FundAccount(s.TestAccount.acc.GetAddress(), sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(100000))))
				msg := testdata.NewTestMsg(s.TestAccount.acc.GetAddress())
				s.Require().NoError(s.txBuilder.SetMsgs(msg))
				s.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(1000))))
				s.txBuilder.SetGasLimit(testGasLimit)
				tx, err := s.CreateTestTx(
					[]cryptotypes.PrivKey{s.TestAccount.priv},
					[]uint64{s.TestAccount.acc.GetAccountNumber()},
					[]uint64{s.TestAccount.acc.GetSequence()},
					s.ctx.ChainID(),
				)
				s.Require().NoError(err)
				return tx
			},
			expErr:    true,
			errSubstr: "blacklisted",
		},
		{
			name: "authz MsgExec with non-blacklisted granter allowed",
			malleate: func() signing.Tx {
				grantee := s.TestAccounts[0]
				granter := s.TestAccounts[1]
				s.FundAccount(grantee.acc.GetAddress(), sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(100000))))
				s.FundAccount(granter.acc.GetAddress(), sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(100000))))

				inner := banktypes.NewMsgSend(
					granter.acc.GetAddress(),
					grantee.acc.GetAddress(),
					sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(1))),
				)
				exec := authz.NewMsgExec(grantee.acc.GetAddress(), []sdk.Msg{inner})
				s.Require().NoError(s.txBuilder.SetMsgs(&exec))
				s.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(1000))))
				s.txBuilder.SetGasLimit(testGasLimit)
				tx, err := s.CreateTestTx(
					[]cryptotypes.PrivKey{grantee.priv},
					[]uint64{grantee.acc.GetAccountNumber()},
					[]uint64{grantee.acc.GetSequence()},
					s.ctx.ChainID(),
				)
				s.Require().NoError(err)
				return tx
			},
			expErr: false,
		},
		{
			name: "authz MsgExec with blacklisted granter rejected",
			malleate: func() signing.Tx {
				grantee := s.TestAccounts[0]
				granter := s.TestAccounts[1]
				s.tfk.AddBlacklist(s.ctx, granter.acc.GetAddress())
				s.FundAccount(grantee.acc.GetAddress(), sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(100000))))

				inner := banktypes.NewMsgSend(
					granter.acc.GetAddress(),
					grantee.acc.GetAddress(),
					sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(1))),
				)
				exec := authz.NewMsgExec(grantee.acc.GetAddress(), []sdk.Msg{inner})
				s.Require().NoError(s.txBuilder.SetMsgs(&exec))
				s.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(1000))))
				s.txBuilder.SetGasLimit(testGasLimit)
				tx, err := s.CreateTestTx(
					[]cryptotypes.PrivKey{grantee.priv},
					[]uint64{grantee.acc.GetAccountNumber()},
					[]uint64{grantee.acc.GetSequence()},
					s.ctx.ChainID(),
				)
				s.Require().NoError(err)
				return tx
			},
			expErr:    true,
			errSubstr: "blacklisted",
		},
		{
			name: "nested authz MsgExec with blacklisted granter rejected",
			malleate: func() signing.Tx {
				outerGrantee := s.TestAccounts[0]
				midGrantee := s.TestAccounts[1]
				priv, _, addr := testdata.KeyTestPubAddr()
				granterAcc := s.ak.NewAccountWithAddress(s.ctx, addr)
				granterAcc.SetAccountNumber(2000)
				s.ak.SetAccount(s.ctx, granterAcc)
				granter := TestAccount{acc: granterAcc, priv: priv}

				s.tfk.AddBlacklist(s.ctx, granter.acc.GetAddress())
				s.FundAccount(outerGrantee.acc.GetAddress(), sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(100000))))

				inner := banktypes.NewMsgSend(
					granter.acc.GetAddress(),
					outerGrantee.acc.GetAddress(),
					sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(1))),
				)
				midExec := authz.NewMsgExec(midGrantee.acc.GetAddress(), []sdk.Msg{inner})
				outerExec := authz.NewMsgExec(outerGrantee.acc.GetAddress(), []sdk.Msg{&midExec})
				s.Require().NoError(s.txBuilder.SetMsgs(&outerExec))
				s.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(1000))))
				s.txBuilder.SetGasLimit(testGasLimit)
				tx, err := s.CreateTestTx(
					[]cryptotypes.PrivKey{outerGrantee.priv},
					[]uint64{outerGrantee.acc.GetAccountNumber()},
					[]uint64{outerGrantee.acc.GetSequence()},
					s.ctx.ChainID(),
				)
				s.Require().NoError(err)
				return tx
			},
			expErr:    true,
			errSubstr: "blacklisted",
		},
		{
			name: "authz MsgExec nesting beyond max depth rejected",
			malleate: func() signing.Tx {
				outer := s.TestAccounts[0]
				mid := s.TestAccounts[1]
				s.FundAccount(outer.acc.GetAddress(), sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(100000))))

				leaf := banktypes.NewMsgSend(
					mid.acc.GetAddress(),
					outer.acc.GetAddress(),
					sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(1))),
				)
				// 4 nested MsgExec layers (> maxAuthzExecDepth=3)
				exec4 := authz.NewMsgExec(mid.acc.GetAddress(), []sdk.Msg{leaf})
				exec3 := authz.NewMsgExec(mid.acc.GetAddress(), []sdk.Msg{&exec4})
				exec2 := authz.NewMsgExec(mid.acc.GetAddress(), []sdk.Msg{&exec3})
				exec1 := authz.NewMsgExec(outer.acc.GetAddress(), []sdk.Msg{&exec2})

				s.Require().NoError(s.txBuilder.SetMsgs(&exec1))
				s.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(1000))))
				s.txBuilder.SetGasLimit(testGasLimit)
				tx, err := s.CreateTestTx(
					[]cryptotypes.PrivKey{outer.priv},
					[]uint64{outer.acc.GetAccountNumber()},
					[]uint64{outer.acc.GetSequence()},
					s.ctx.ChainID(),
				)
				s.Require().NoError(err)
				return tx
			},
			expErr:    true,
			errSubstr: "nesting exceeds max depth",
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.SetupTest()
			handler := sdk.ChainAnteDecorators(
				txfeesante.NewBlacklistDecorator(s.tfk, s.app.AppCodec()),
			)
			tx := tc.malleate()
			_, err := handler(s.ctx, tx, false)
			if !tc.expErr {
				s.Require().NoError(err)
				return
			}
			s.Require().Error(err)
			if tc.errSubstr != "" {
				s.Require().Contains(err.Error(), tc.errSubstr)
			}
		})
	}
}
