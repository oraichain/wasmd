package ante_test

import (
	"time"

	"cosmossdk.io/math"
	"cosmossdk.io/x/feegrant"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
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
