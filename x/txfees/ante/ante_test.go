package ante_test

import (
	"cosmossdk.io/math"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
)

var testGasLimit uint64 = 200_000

func (s *AnteTestSuite) TestGlobalFeeMinimumGasFeeAnteHandler() {

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
				FundAccount(s.ctx, s.bk, s.TestAccount.acc.GetAddress(), sdk.NewCoins(sdk.NewCoin("orai", math.NewInt(100000))))
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

				FundAccount(s.ctx, s.bk, s.TestAccount.acc.GetAddress(), sdk.NewCoins(sdk.NewCoin("usdai", math.NewInt(100000))))
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

				FundAccount(s.ctx, s.bk, s.TestAccount.acc.GetAddress(), sdk.NewCoins(sdk.NewCoin("usdai", math.NewInt(100000))))
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

				FundAccount(s.ctx, s.bk, s.TestAccount.acc.GetAddress(), sdk.NewCoins(sdk.NewCoin("usdai", math.NewInt(100000))))
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

				FundAccount(s.ctx, s.bk, s.TestAccount.acc.GetAddress(), sdk.NewCoins(sdk.NewCoin("usdc", math.NewInt(100000))))
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
