package v10_test

import (
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/suite"

	wasmApp "github.com/CosmWasm/wasmd/app"
	v10 "github.com/CosmWasm/wasmd/app/upgrades/v05014"
	appconfig "github.com/CosmWasm/wasmd/cmd/config"
	sdk "github.com/cosmos/cosmos-sdk/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
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
