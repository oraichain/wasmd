package v10_test

import (
	"encoding/json"
	"testing"
	"time"

	_ "embed"

	sdkmath "cosmossdk.io/math"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/suite"

	wasmApp "github.com/CosmWasm/wasmd/app"
	v10 "github.com/CosmWasm/wasmd/app/upgrades/v05014"
	appconfig "github.com/CosmWasm/wasmd/cmd/config"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
)

//go:embed testdata/cw20_base.wasm
var cw20BaseWasm []byte

// Instantiate2 fixtures for localfork mock RecoveryAssets (fixMsg=false).
const (
	recoveryCW20DeployerMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	recoveryCW20DeployerAddr     = "orai19rl4cm2hmr8afy4kldpxz3fka4jguq0a0nm77x"
	recoveryCW20Salt1            = "localfork-cw20-v1"
	recoveryCW20Salt2            = "localfork-cw20-v2"
)

type UpgradeTestSuite struct {
	suite.Suite

	App *wasmApp.WasmApp
	Ctx sdk.Context

	defaultRecoveryAssets      []string
	defaultRecoveryFromAddress []string
}

func TestUpgradeTestSuite(t *testing.T) {
	suite.Run(t, new(UpgradeTestSuite))
}

func (s *UpgradeTestSuite) SetupTest() {
	wasmApp.SetSDKConfig()
	s.App = wasmApp.Setup(s.T())
	s.Ctx = s.App.NewContextLegacy(false, cmtproto.Header{
		Height: 1,
		Time:   time.Now().UTC(),
	})
	if s.defaultRecoveryAssets == nil {
		s.defaultRecoveryAssets = append([]string{}, v10.RecoveryAssets...)
	}
	if s.defaultRecoveryFromAddress == nil {
		s.defaultRecoveryFromAddress = append([]string{}, v10.RecoveryFromAddress...)
	}
	// Burn-only tests: skip CW20 rescue (package defaults are Instantiate2 mock addrs).
	v10.RecoveryAssets = nil
	v10.RecoveryFromAddress = nil
	v10.RecoveryAddress = ""
}

func (s *UpgradeTestSuite) BeginNewBlock() {
	newHeight := s.Ctx.BlockHeight() + 1
	newTime := s.Ctx.BlockTime().Add(time.Second)

	s.Ctx = s.Ctx.WithBlockHeight(newHeight).WithBlockTime(newTime)
	_, err := s.App.BeginBlocker(s.Ctx)
	s.Require().NoError(err)
	s.Ctx = s.Ctx.WithBlockHeader(cmtproto.Header{Height: newHeight, Time: newTime})
}

func (s *UpgradeTestSuite) fund(addr sdk.AccAddress, amount int64) {
	coins := sdk.NewCoins(sdk.NewCoin(appconfig.MinimalDenom, sdkmath.NewInt(amount)))
	s.Require().NoError(s.App.BankKeeper.MintCoins(s.Ctx, minttypes.ModuleName, coins))
	s.Require().NoError(s.App.BankKeeper.SendCoinsFromModuleToAccount(s.Ctx, minttypes.ModuleName, addr, coins))
}

func (s *UpgradeTestSuite) balance(addr sdk.AccAddress) sdkmath.Int {
	return s.App.BankKeeper.GetBalance(s.Ctx, addr, appconfig.MinimalDenom).Amount
}

func (s *UpgradeTestSuite) TestForkBeginBlockerSucceeds() {
	s.Ctx = s.Ctx.WithBlockHeight(v10.ForkHeight - 2)
	s.BeginNewBlock() // ForkHeight-1
	s.Require().NotPanics(func() {
		s.BeginNewBlock() // ForkHeight — burn + activate
	})
}

func (s *UpgradeTestSuite) TestBlacklistBurnAtForkHeightDespiteInitList() {
	victim := sdk.AccAddress("blacklist-victim0001") // 20 bytes
	other := sdk.AccAddress("blacklist-other000002")

	const victimFund int64 = 5_000_000
	s.fund(victim, victimFund)
	s.fund(other, victimFund)
	s.Require().True(s.balance(victim).Equal(sdkmath.NewInt(victimFund)))

	// Address is on the blacklist set before burn (mirrors init()/ActivateSendBlacklist).
	v10.BlacklistAddresses = []string{victim.String()}
	wasmApp.AddSendBlacklistAddress(victim.String())
	s.Require().True(wasmApp.IsSendBlacklisted(victim.String()))

	// Burn at ForkHeight must succeed: restriction is height-gated (<= ForkHeight => no-op).
	s.Ctx = s.Ctx.WithBlockHeight(v10.ForkHeight)
	keepers := s.App.GetUpgradeKeepers()
	s.Require().NotPanics(func() {
		v10.RunForkLogic(s.Ctx, &keepers)
	})

	s.Require().True(s.balance(victim).IsZero(), "ORAI on blacklisted addr must be burned at fork")
	s.Require().True(s.balance(other).Equal(sdkmath.NewInt(victimFund)), "non-blacklist balances unchanged")

	bl, err := keepers.TxFeesKeeper.IsBlacklisted(s.Ctx, victim)
	s.Require().NoError(err)
	s.Require().True(bl, "victim must be in txfees blacklist store after fork")
	blOther, err := keepers.TxFeesKeeper.IsBlacklisted(s.Ctx, other)
	s.Require().NoError(err)
	s.Require().False(blOther, "non-blacklist addr must not be in txfees store")
}

func (s *UpgradeTestSuite) TestBlacklistRestrictionOnlyAfterForkBlock() {
	victim := sdk.AccAddress("blacklist-victim0003")
	other := sdk.AccAddress("blacklist-other000004")

	s.fund(other, 10_000_000)
	v10.BlacklistAddresses = []string{victim.String()}
	wasmApp.ActivateSendBlacklist()
	s.Require().True(wasmApp.IsSendBlacklisted(victim.String()))

	one := sdk.NewCoins(sdk.NewCoin(appconfig.MinimalDenom, sdkmath.NewInt(1)))

	// At ForkHeight: sends to/from blacklist still allowed.
	s.Ctx = s.Ctx.WithBlockHeight(v10.ForkHeight)
	s.Require().NoError(s.App.BankKeeper.SendCoins(s.Ctx, other, victim, one))
	s.Require().True(s.balance(victim).Equal(sdkmath.NewInt(1)))

	s.Require().NoError(s.App.BankKeeper.SendCoins(s.Ctx, victim, other, one))
	s.Require().True(s.balance(victim).IsZero())

	// After fork block: in and out blocked.
	s.Ctx = s.Ctx.WithBlockHeight(v10.ForkHeight + 1)
	s.Require().NoError(s.App.BankKeeper.SendCoins(s.Ctx, other, other, one)) // control: normal send OK

	s.Require().Error(s.App.BankKeeper.SendCoins(s.Ctx, other, victim, one), "in must be blocked after fork")
	s.Require().Error(s.App.BankKeeper.SendCoins(s.Ctx, victim, other, one), "out must be blocked after fork")
}

func (s *UpgradeTestSuite) TestBeginBlockForkBurnsThenBlocksNextHeight() {
	victim := sdk.AccAddress("blacklist-victim0005")
	other := sdk.AccAddress("blacklist-other000006")

	const victimFund int64 = 3_000_000
	s.fund(victim, victimFund)
	s.fund(other, victimFund)

	v10.BlacklistAddresses = []string{victim.String()}
	wasmApp.AddSendBlacklistAddress(victim.String())

	// Drive real BeginBlocker path through fork height.
	s.Ctx = s.Ctx.WithBlockHeight(v10.ForkHeight - 1)
	s.BeginNewBlock() // == ForkHeight: RunForkLogic (burn)
	s.Require().True(s.balance(victim).IsZero(), "burn via BeginBlockForks")

	// Same fork height: restriction still inactive.
	one := sdk.NewCoins(sdk.NewCoin(appconfig.MinimalDenom, sdkmath.NewInt(1)))
	s.Require().NoError(s.App.BankKeeper.SendCoins(s.Ctx, other, victim, one))

	// Next block: restriction active.
	s.BeginNewBlock() // ForkHeight+1
	s.Require().Error(s.App.BankKeeper.SendCoins(s.Ctx, other, victim, one))
	s.Require().Error(s.App.BankKeeper.SendCoins(s.Ctx, victim, other, one))
}

type cw20InstantiateMsg struct {
	Name            string      `json:"name"`
	Symbol          string      `json:"symbol"`
	Decimals        uint8       `json:"decimals"`
	InitialBalances []cw20Coin  `json:"initial_balances"`
	Mint            *cw20Minter `json:"mint,omitempty"`
}

type cw20Coin struct {
	Address string `json:"address"`
	Amount  string `json:"amount"`
}

type cw20Minter struct {
	Minter string `json:"minter"`
}

type cw20MintMsg struct {
	Mint *struct {
		Recipient string `json:"recipient"`
		Amount    string `json:"amount"`
	} `json:"mint"`
}

type cw20BalanceQuery struct {
	Balance *struct {
		Address string `json:"address"`
	} `json:"balance"`
}

type cw20BalanceResponse struct {
	Balance string `json:"balance"`
}

func (s *UpgradeTestSuite) cw20Balance(contract, addr sdk.AccAddress) string {
	q, err := json.Marshal(cw20BalanceQuery{
		Balance: &struct {
			Address string `json:"address"`
		}{Address: addr.String()},
	})
	s.Require().NoError(err)
	bz, err := s.App.WasmKeeper.QuerySmart(s.Ctx, contract, q)
	s.Require().NoError(err)
	var resp cw20BalanceResponse
	s.Require().NoError(json.Unmarshal(bz, &resp))
	return resp.Balance
}

func (s *UpgradeTestSuite) TestCw20RescueViaForkExecute() {
	s.Require().NotEmpty(cw20BaseWasm, "embedded cw20_base.wasm")
	s.Require().Len(s.defaultRecoveryAssets, 2, "hardcoded Instantiate2 RecoveryAssets")

	deployer, err := sdk.AccAddressFromBech32(recoveryCW20DeployerAddr)
	s.Require().NoError(err)
	s.Require().NotEmpty(s.defaultRecoveryFromAddress)
	victim, err := sdk.AccAddressFromBech32(s.defaultRecoveryFromAddress[0])
	s.Require().NoError(err)
	recipient := sdk.AccAddress("cw20-rescue-to-addr01")
	const cw20Amt = "1000000000"
	const cw20Amt2 = "500000000"

	s.fund(deployer, 10_000_000)
	v10.BlacklistAddresses = []string{victim.String()}
	wasmApp.AddSendBlacklistAddress(victim.String())

	keepers := s.App.GetUpgradeKeepers()
	s.Require().NotNil(keepers.ContractKeeper)

	codeID, checksum, err := keepers.ContractKeeper.Create(s.Ctx, deployer, cw20BaseWasm, nil)
	s.Require().NoError(err)

	// Address must match hardcoded RecoveryAssets (Instantiate2, fixMsg=false).
	pred1 := wasmkeeper.BuildContractAddressPredictable(checksum, deployer, []byte(recoveryCW20Salt1), []byte{})
	pred2 := wasmkeeper.BuildContractAddressPredictable(checksum, deployer, []byte(recoveryCW20Salt2), []byte{})
	s.Require().Equal(s.defaultRecoveryAssets[0], pred1.String())
	s.Require().Equal(s.defaultRecoveryAssets[1], pred2.String())

	deploy := func(salt, label, amount string, want sdk.AccAddress) sdk.AccAddress {
		initMsg, err := json.Marshal(cw20InstantiateMsg{
			Name:            "ForkTest",
			Symbol:          "FRK",
			Decimals:        6,
			InitialBalances: []cw20Coin{},
			Mint:            &cw20Minter{Minter: deployer.String()},
		})
		s.Require().NoError(err)
		contractAddr, _, err := keepers.ContractKeeper.Instantiate2(
			s.Ctx, codeID, deployer, deployer, initMsg, label, nil, []byte(salt), false,
		)
		s.Require().NoError(err)
		s.Require().Equal(want.String(), contractAddr.String())
		mintMsg, err := json.Marshal(cw20MintMsg{
			Mint: &struct {
				Recipient string `json:"recipient"`
				Amount    string `json:"amount"`
			}{Recipient: victim.String(), Amount: amount},
		})
		s.Require().NoError(err)
		_, err = keepers.ContractKeeper.Execute(s.Ctx, contractAddr, deployer, mintMsg, nil)
		s.Require().NoError(err)
		s.Require().Equal(amount, s.cw20Balance(contractAddr, victim))
		return contractAddr
	}

	c1 := deploy(recoveryCW20Salt1, "localfork-cw20-a", cw20Amt, pred1)
	c2 := deploy(recoveryCW20Salt2, "localfork-cw20-b", cw20Amt2, pred2)

	v10.RecoveryAssets = append([]string{}, s.defaultRecoveryAssets...)
	v10.RecoveryFromAddress = append([]string{}, s.defaultRecoveryFromAddress...)
	v10.RecoveryAddress = recipient.String()

	s.Ctx = s.Ctx.WithBlockHeight(v10.ForkHeight)
	keepers = s.App.GetUpgradeKeepers()
	s.Require().NotPanics(func() {
		v10.RunForkLogic(s.Ctx, &keepers)
	})

	s.Require().Equal("0", s.cw20Balance(c1, victim))
	s.Require().Equal(cw20Amt, s.cw20Balance(c1, recipient))
	s.Require().Equal("0", s.cw20Balance(c2, victim))
	s.Require().Equal(cw20Amt2, s.cw20Balance(c2, recipient))
}
