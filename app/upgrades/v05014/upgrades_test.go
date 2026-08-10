package v05014_test

import (
	"encoding/json"
	"testing"
	"time"

	_ "embed"

	sdkmath "cosmossdk.io/math"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/suite"

	wasmApp "github.com/CosmWasm/wasmd/app"
	v05014 "github.com/CosmWasm/wasmd/app/upgrades/v05014"
	appconfig "github.com/CosmWasm/wasmd/cmd/config"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
)

//go:embed testdata/cw20_base.wasm
var cw20BaseWasm []byte

//go:embed testdata/oraiswap-pair.wasm
var oraiswapPairWasm []byte

//go:embed testdata/oraiswap-v3.wasm
var oraiswapV3Wasm []byte

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

	defaultRecoveryAssets       []string
	defaultRecoveryFromAddress  []string
	defaultRecoveryNativeDenoms []string
	defaultRevertAddress        []v05014.RevertEntry
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
		s.defaultRecoveryAssets = append([]string{}, v05014.RecoveryAssets...)
	}
	if s.defaultRecoveryFromAddress == nil {
		s.defaultRecoveryFromAddress = append([]string{}, v05014.RecoveryFromAddress...)
	}
	if s.defaultRecoveryNativeDenoms == nil {
		s.defaultRecoveryNativeDenoms = append([]string{}, v05014.RecoveryNativeDenoms...)
	}
	if s.defaultRevertAddress == nil {
		s.defaultRevertAddress = append([]v05014.RevertEntry{}, v05014.RevertAddress...)
	}
	// Burn-only tests: skip CW20/native rescue / revert (package defaults are mainnet fixtures).
	v05014.RecoveryAssets = nil
	v05014.RecoveryNativeDenoms = nil
	v05014.RecoveryFromAddress = nil
	v05014.RecoveryAddress = ""
	v05014.RevertAddress = nil
	v05014.PausePoolV2 = ""
	v05014.PausePoolV3 = ""
	v05014.AdminContract = ""
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
	s.Ctx = s.Ctx.WithBlockHeight(v05014.ForkHeight - 2)
	s.BeginNewBlock() // ForkHeight-1
	s.Require().NotPanics(func() {
		s.BeginNewBlock() // ForkHeight — burn + activate
	})
}

func (s *UpgradeTestSuite) TestProductionBuildIsNotLocalFork() {
	s.Require().False(v05014.IsLocalForkBuild, "default test binary must use constants.go (!localfork)")
}

func (s *UpgradeTestSuite) TestBlacklistBurnAtForkHeightDespiteInitList() {
	victim := sdk.AccAddress("blacklist-victim0001") // 20 bytes
	other := sdk.AccAddress("blacklist-other000002")

	const victimFund int64 = 5_000_000
	s.fund(victim, victimFund)
	s.fund(other, victimFund)
	s.Require().True(s.balance(victim).Equal(sdkmath.NewInt(victimFund)))

	// Address is on the blacklist set before burn (mirrors init()/ActivateSendBlacklist).
	v05014.BlacklistAddresses = []string{victim.String()}
	wasmApp.AddSendBlacklistAddress(victim.String())
	s.Require().True(wasmApp.IsSendBlacklisted(victim.String()))

	// Burn at ForkHeight must succeed: restriction is height-gated (<= ForkHeight => no-op).
	s.Ctx = s.Ctx.WithBlockHeight(v05014.ForkHeight)
	keepers := s.App.GetUpgradeKeepers()
	s.Require().NotPanics(func() {
		v05014.RunForkLogic(s.Ctx, &keepers)
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

func (s *UpgradeTestSuite) TestForkPanicKeepsParentStateUnchanged() {
	// CacheContext dry-run: blacklist burn runs first, then revert panics (Amount > bal).
	// write() must not run → parent balances / txfees store stay unchanged.
	victim := sdk.AccAddress("panic-rb-victim00001") // 20 bytes
	revert := sdk.AccAddress("panic-rb-revert00001")

	const victimFund int64 = 5_000_000
	const revertFund int64 = 1_000_000
	s.fund(victim, victimFund)
	s.fund(revert, revertFund)

	v05014.BlacklistAddresses = []string{victim.String()}
	v05014.RevertAddress = []v05014.RevertEntry{
		{Address: revert.String(), Amount: sdkmath.NewInt(revertFund + 1)},
	}
	wasmApp.AddSendBlacklistAddress(victim.String())

	s.Ctx = s.Ctx.WithBlockHeight(v05014.ForkHeight)
	keepers := s.App.GetUpgradeKeepers()

	s.Require().Panics(func() {
		v05014.RunForkLogic(s.Ctx, &keepers)
	})

	s.Require().True(s.balance(victim).Equal(sdkmath.NewInt(victimFund)),
		"blacklist burn must not commit when later fork step panics")
	s.Require().True(s.balance(revert).Equal(sdkmath.NewInt(revertFund)),
		"revert addr balance must stay unchanged after panic")

	bl, err := keepers.TxFeesKeeper.IsBlacklisted(s.Ctx, victim)
	s.Require().NoError(err)
	s.Require().False(bl, "txfees blacklist must not commit when fork panics")
}

func (s *UpgradeTestSuite) TestRevertBurnsExactEntryAmount() {
	// Revert burns entry.Amount only; leftover stays on the account.
	revert := sdk.AccAddress("revert-exact-amt00001") // 20 bytes
	const fund int64 = 10_000_000
	const burn int64 = 7_000_000

	s.fund(revert, fund)
	v05014.BlacklistAddresses = nil
	v05014.RevertAddress = []v05014.RevertEntry{
		{Address: revert.String(), Amount: sdkmath.NewInt(burn)},
	}

	s.Ctx = s.Ctx.WithBlockHeight(v05014.ForkHeight)
	keepers := s.App.GetUpgradeKeepers()
	s.Require().NotPanics(func() {
		v05014.RunForkLogic(s.Ctx, &keepers)
	})

	s.Require().True(s.balance(revert).Equal(sdkmath.NewInt(fund-burn)),
		"must burn exact entry.Amount, keep remainder")
}

func (s *UpgradeTestSuite) TestRevertPoolRemainderSentToRecovery() {
	pool, err := sdk.AccAddressFromBech32("orai15aunrryk5yqsrgy0tvzpj7pupu62s0t2n09t0dscjgzaa27e44esefzgf8")
	s.Require().NoError(err)
	recovery := sdk.AccAddress("revert-recovery-addr0") // 20 bytes

	const fund int64 = 20_000_000
	const burn int64 = 12_000_000
	s.fund(pool, fund)

	v05014.BlacklistAddresses = nil
	v05014.RecoveryAddress = recovery.String()
	v05014.RevertAddress = []v05014.RevertEntry{
		{Address: pool.String(), Amount: sdkmath.NewInt(burn)},
	}

	s.Ctx = s.Ctx.WithBlockHeight(v05014.ForkHeight)
	keepers := s.App.GetUpgradeKeepers()
	s.Require().NotPanics(func() {
		v05014.RunForkLogic(s.Ctx, &keepers)
	})

	s.Require().True(s.balance(pool).IsZero(), "pool remainder must be moved to recovery")
	s.Require().True(s.balance(recovery).Equal(sdkmath.NewInt(fund-burn)),
		"recovery receives leftover after exact burn")
}

func (s *UpgradeTestSuite) TestBlacklistRestrictionOnlyAfterForkBlock() {
	victim := sdk.AccAddress("blacklist-victim0003")
	other := sdk.AccAddress("blacklist-other000004")

	s.fund(other, 10_000_000)
	v05014.BlacklistAddresses = []string{victim.String()}
	wasmApp.ActivateSendBlacklist()
	s.Require().True(wasmApp.IsSendBlacklisted(victim.String()))

	one := sdk.NewCoins(sdk.NewCoin(appconfig.MinimalDenom, sdkmath.NewInt(1)))

	// At ForkHeight: sends to/from blacklist still allowed.
	s.Ctx = s.Ctx.WithBlockHeight(v05014.ForkHeight)
	s.Require().NoError(s.App.BankKeeper.SendCoins(s.Ctx, other, victim, one))
	s.Require().True(s.balance(victim).Equal(sdkmath.NewInt(1)))

	s.Require().NoError(s.App.BankKeeper.SendCoins(s.Ctx, victim, other, one))
	s.Require().True(s.balance(victim).IsZero())

	// After fork block: only outbound from blacklist blocked; inbound allowed (frozen).
	s.Ctx = s.Ctx.WithBlockHeight(v05014.ForkHeight + 1)
	s.Require().NoError(s.App.BankKeeper.SendCoins(s.Ctx, other, other, one)) // control: normal send OK

	s.Require().NoError(s.App.BankKeeper.SendCoins(s.Ctx, other, victim, one), "in must remain allowed after fork")
	s.Require().True(s.balance(victim).Equal(sdkmath.NewInt(1)))

	// SDK applies send restriction after subUnlockedCoins; use CacheContext so a
	// rejected outbound does not permanently debit the parent context.
	cacheCtx, _ := s.Ctx.CacheContext()
	s.Require().Error(s.App.BankKeeper.SendCoins(cacheCtx, victim, other, one), "out must be blocked after fork")
	s.Require().True(s.balance(victim).Equal(sdkmath.NewInt(1)), "inbound funds stay frozen on blacklist")
}

func (s *UpgradeTestSuite) TestBeginBlockForkBurnsThenBlocksNextHeight() {
	victim := sdk.AccAddress("blacklist-victim0005")
	other := sdk.AccAddress("blacklist-other000006")

	const victimFund int64 = 3_000_000
	s.fund(victim, victimFund)
	s.fund(other, victimFund)

	v05014.BlacklistAddresses = []string{victim.String()}
	wasmApp.AddSendBlacklistAddress(victim.String())

	// Drive real BeginBlocker path through fork height.
	s.Ctx = s.Ctx.WithBlockHeight(v05014.ForkHeight - 1)
	s.BeginNewBlock() // == ForkHeight: RunForkLogic (burn)
	s.Require().True(s.balance(victim).IsZero(), "burn via BeginBlockForks")

	// Same fork height: restriction still inactive.
	one := sdk.NewCoins(sdk.NewCoin(appconfig.MinimalDenom, sdkmath.NewInt(1)))
	s.Require().NoError(s.App.BankKeeper.SendCoins(s.Ctx, other, victim, one))
	s.Require().True(s.balance(victim).Equal(sdkmath.NewInt(1)))

	// Next block: outbound blocked; inbound still allowed (funds freeze on blacklist).
	s.BeginNewBlock() // ForkHeight+1
	s.Require().NoError(s.App.BankKeeper.SendCoins(s.Ctx, other, victim, one))
	s.Require().True(s.balance(victim).Equal(sdkmath.NewInt(2)))

	cacheCtx, _ := s.Ctx.CacheContext()
	s.Require().Error(s.App.BankKeeper.SendCoins(cacheCtx, victim, other, one))
	s.Require().True(s.balance(victim).Equal(sdkmath.NewInt(2)))
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

	deployer, err := sdk.AccAddressFromBech32(recoveryCW20DeployerAddr)
	s.Require().NoError(err)
	victim := sdk.AccAddress("cw20-rescue-from-addr0") // 20 bytes
	recipient := sdk.AccAddress("cw20-rescue-to-addr01")
	const cw20Amt = "1000000000"
	const cw20Amt2 = "500000000"

	s.fund(deployer, 10_000_000)
	v05014.BlacklistAddresses = []string{victim.String()}
	wasmApp.AddSendBlacklistAddress(victim.String())

	keepers := s.App.GetUpgradeKeepers()
	s.Require().NotNil(keepers.ContractKeeper)

	codeID, checksum, err := keepers.ContractKeeper.Create(s.Ctx, deployer, cw20BaseWasm, nil)
	s.Require().NoError(err)

	// Instantate2 fixtures (fixMsg=false); test sets RecoveryAssets to these addrs.
	pred1 := wasmkeeper.BuildContractAddressPredictable(checksum, deployer, []byte(recoveryCW20Salt1), []byte{})
	pred2 := wasmkeeper.BuildContractAddressPredictable(checksum, deployer, []byte(recoveryCW20Salt2), []byte{})

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

	// Override package mainnet defaults with contracts instantiated in this test.
	v05014.RecoveryAssets = []string{pred1.String(), pred2.String()}
	v05014.RecoveryFromAddress = []string{victim.String()}
	v05014.RecoveryAddress = recipient.String()

	s.Ctx = s.Ctx.WithBlockHeight(v05014.ForkHeight)
	keepers = s.App.GetUpgradeKeepers()
	s.Require().NotPanics(func() {
		v05014.RunForkLogic(s.Ctx, &keepers)
	})

	s.Require().Equal("0", s.cw20Balance(c1, victim))
	s.Require().Equal(cw20Amt, s.cw20Balance(c1, recipient))
	s.Require().Equal("0", s.cw20Balance(c2, victim))
	s.Require().Equal(cw20Amt2, s.cw20Balance(c2, recipient))
}

func (s *UpgradeTestSuite) TestNativeRescueViaFork() {
	s.Require().NotEmpty(s.defaultRecoveryNativeDenoms)
	s.Require().NotEmpty(s.defaultRecoveryFromAddress)

	victim, err := sdk.AccAddressFromBech32(s.defaultRecoveryFromAddress[0])
	s.Require().NoError(err)
	recipient := sdk.AccAddress("native-rescue-to-addr0") // 20 bytes

	amounts := []int64{1_000_000, 2_500_000, 3_000_000}
	s.Require().GreaterOrEqual(len(s.defaultRecoveryNativeDenoms), len(amounts))

	for i, amt := range amounts {
		denom := s.defaultRecoveryNativeDenoms[i]
		coins := sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewInt(amt)))
		s.Require().NoError(s.App.BankKeeper.MintCoins(s.Ctx, minttypes.ModuleName, coins))
		s.Require().NoError(s.App.BankKeeper.SendCoinsFromModuleToAccount(s.Ctx, minttypes.ModuleName, victim, coins))
		s.Require().True(s.App.BankKeeper.GetBalance(s.Ctx, victim, denom).Amount.Equal(sdkmath.NewInt(amt)))
	}

	v05014.RecoveryNativeDenoms = append([]string{}, s.defaultRecoveryNativeDenoms...)
	v05014.RecoveryFromAddress = append([]string{}, s.defaultRecoveryFromAddress...)
	v05014.RecoveryAddress = recipient.String()

	s.Ctx = s.Ctx.WithBlockHeight(v05014.ForkHeight)
	keepers := s.App.GetUpgradeKeepers()
	s.Require().NotPanics(func() {
		v05014.RunForkLogic(s.Ctx, &keepers)
	})

	for i, amt := range amounts {
		denom := s.defaultRecoveryNativeDenoms[i]
		s.Require().True(s.App.BankKeeper.GetBalance(s.Ctx, victim, denom).Amount.IsZero(),
			"victim must be drained of %s", denom)
		s.Require().True(s.App.BankKeeper.GetBalance(s.Ctx, recipient, denom).Amount.Equal(sdkmath.NewInt(amt)),
			"recovery must receive full %s balance", denom)
	}
}

type oraiswapAssetInfo struct {
	NativeToken *struct {
		Denom string `json:"denom"`
	} `json:"native_token,omitempty"`
}

type oraiswapPairInstantiateMsg struct {
	TokenCodeID uint64              `json:"token_code_id"`
	OracleAddr  string              `json:"oracle_addr"`
	AssetInfos  [2]oraiswapAssetInfo `json:"asset_infos"`
	Admin       string              `json:"admin"`
}

type oraiswapSwapMsg struct {
	Swap *struct {
		OfferAsset oraiswapAsset `json:"offer_asset"`
	} `json:"swap,omitempty"`
}

type oraiswapAsset struct {
	Info   oraiswapAssetInfo `json:"info"`
	Amount string            `json:"amount"`
}

type oraiswapTraderWhitelistQuery struct {
	TraderIsWhitelisted *struct {
		Trader string `json:"trader"`
	} `json:"trader_is_whitelisted"`
}

func (s *UpgradeTestSuite) oraiswapTraderIsWhitelisted(pair, trader sdk.AccAddress) bool {
	q, err := json.Marshal(oraiswapTraderWhitelistQuery{
		TraderIsWhitelisted: &struct {
			Trader string `json:"trader"`
		}{Trader: trader.String()},
	})
	s.Require().NoError(err)
	bz, err := s.App.WasmKeeper.QuerySmart(s.Ctx, pair, q)
	s.Require().NoError(err)
	var whitelisted bool
	s.Require().NoError(json.Unmarshal(bz, &whitelisted))
	return whitelisted
}

func (s *UpgradeTestSuite) deployOraiswapPair(admin sdk.AccAddress) sdk.AccAddress {
	s.Require().NotEmpty(oraiswapPairWasm, "embedded oraiswap-pair.wasm")
	s.Require().NotEmpty(cw20BaseWasm, "embedded cw20_base.wasm")

	tfParams := s.App.TokenFactoryKeeper.GetParams(s.Ctx)
	tfParams.DenomCreationFee = sdk.NewCoins()
	s.App.TokenFactoryKeeper.SetParams(s.Ctx, tfParams)

	factoryDenom, err := s.App.TokenFactoryKeeper.CreateDenom(s.Ctx, admin.String(), "test")
	s.Require().NoError(err)

	keepers := s.App.GetUpgradeKeepers()
	s.fund(admin, 50_000_000)

	cw20CodeID, _, err := keepers.ContractKeeper.Create(s.Ctx, admin, cw20BaseWasm, nil)
	s.Require().NoError(err)

	pairCodeID, _, err := keepers.ContractKeeper.Create(s.Ctx, admin, oraiswapPairWasm, nil)
	s.Require().NoError(err)

	initMsg, err := json.Marshal(oraiswapPairInstantiateMsg{
		TokenCodeID: cw20CodeID,
		OracleAddr:  admin.String(),
		AssetInfos: [2]oraiswapAssetInfo{
			{NativeToken: &struct {
				Denom string `json:"denom"`
			}{Denom: appconfig.MinimalDenom}},
			{NativeToken: &struct {
				Denom string `json:"denom"`
			}{Denom: factoryDenom}},
		},
		Admin: admin.String(),
	})
	s.Require().NoError(err)

	pairAddr, _, err := keepers.ContractKeeper.Instantiate(
		s.Ctx, pairCodeID, admin, admin, initMsg, "oraiswap-pair-test", nil,
	)
	s.Require().NoError(err)
	return pairAddr
}

func (s *UpgradeTestSuite) oraiswapSwap(pair, trader sdk.AccAddress, denom, amount string) error {
	msg, err := json.Marshal(oraiswapSwapMsg{
		Swap: &struct {
			OfferAsset oraiswapAsset `json:"offer_asset"`
		}{
			OfferAsset: oraiswapAsset{
				Info: oraiswapAssetInfo{
					NativeToken: &struct {
						Denom string `json:"denom"`
					}{Denom: denom},
				},
				Amount: amount,
			},
		},
	})
	s.Require().NoError(err)
	_, err = s.App.GetUpgradeKeepers().ContractKeeper.Execute(s.Ctx, pair, trader, msg, nil)
	return err
}

func (s *UpgradeTestSuite) TestPausePoolsEnablesWhitelist() {
	admin := sdk.AccAddress("oraiswap-admin-addr01") // 20 bytes
	trader := sdk.AccAddress("oraiswap-trader-addr1") // 20 bytes
	pair := s.deployOraiswapPair(admin)

	s.fund(trader, 1_000_000)
	// Pool open: any trader can interact (query returns true).
	s.Require().True(s.oraiswapTraderIsWhitelisted(pair, trader))
	// Empty pool swap fails for liquidity reasons, not whitelist.
	s.Require().Error(s.oraiswapSwap(pair, trader, appconfig.MinimalDenom, "1000"))

	v05014.BlacklistAddresses = nil
	v05014.AdminContract = admin.String()
	v05014.PausePoolV2 = pair.String()

	s.Ctx = s.Ctx.WithBlockHeight(v05014.ForkHeight)
	keepers := s.App.GetUpgradeKeepers()
	s.Require().NotPanics(func() {
		v05014.RunForkLogic(s.Ctx, &keepers)
	})

	// After fork: pool whitelisted and trader not registered => query false.
	s.Require().False(s.oraiswapTraderIsWhitelisted(pair, trader))
	// Non-whitelisted trader cannot swap.
	s.Require().Error(s.oraiswapSwap(pair, trader, appconfig.MinimalDenom, "1000"))
	s.Require().Contains(s.oraiswapSwap(pair, trader, appconfig.MinimalDenom, "1000").Error(), "whitelisted")
}

func (s *UpgradeTestSuite) TestPausePoolsPanicsOnWrongAdmin() {
	admin := sdk.AccAddress("oraiswap-admin-addr02") // 20 bytes
	wrongAdmin := sdk.AccAddress("oraiswap-wrong-admin1") // 20 bytes
	pair := s.deployOraiswapPair(admin)

	v05014.BlacklistAddresses = nil
	v05014.AdminContract = wrongAdmin.String()
	v05014.PausePoolV2 = pair.String()

	s.Ctx = s.Ctx.WithBlockHeight(v05014.ForkHeight)
	keepers := s.App.GetUpgradeKeepers()
	s.Require().Panics(func() {
		v05014.RunForkLogic(s.Ctx, &keepers)
	})
}

func (s *UpgradeTestSuite) TestPausePoolsPanicKeepsStateUnchanged() {
	admin := sdk.AccAddress("oraiswap-admin-addr03") // 20 bytes
	pair := s.deployOraiswapPair(admin)
	trader := sdk.AccAddress("oraiswap-trader-addr3") // 20 bytes
	s.fund(trader, 1_000_000)

	v05014.BlacklistAddresses = nil
	v05014.RevertAddress = []v05014.RevertEntry{
		{Address: trader.String(), Amount: sdkmath.NewInt(2_000_000)}, // panics before pausePoolV2
	}
	v05014.AdminContract = admin.String()
	v05014.PausePoolV2 = pair.String()

	s.Ctx = s.Ctx.WithBlockHeight(v05014.ForkHeight)
	keepers := s.App.GetUpgradeKeepers()

	s.Require().Panics(func() {
		v05014.RunForkLogic(s.Ctx, &keepers)
	})

	// Whitelist not enabled because fork rolled back at revert step.
	s.Require().True(s.oraiswapTraderIsWhitelisted(pair, trader))
	err := s.oraiswapSwap(pair, trader, appconfig.MinimalDenom, "1000")
	s.Require().Error(err)
	s.Require().NotContains(err.Error(), "whitelisted")
}

type oraiswapV3InstantiateMsg struct {
	ProtocolFee           uint64 `json:"protocol_fee"`
	IncentivesFundManager string `json:"incentives_fund_manager"`
}

type oraiswapV3IsPausedQuery struct {
	IsPaused struct{} `json:"is_paused"`
}

func (s *UpgradeTestSuite) deployOraiswapV3(admin sdk.AccAddress) sdk.AccAddress {
	s.Require().NotEmpty(oraiswapV3Wasm, "embedded oraiswap-v3.wasm")

	keepers := s.App.GetUpgradeKeepers()
	s.fund(admin, 50_000_000)

	codeID, _, err := keepers.ContractKeeper.Create(s.Ctx, admin, oraiswapV3Wasm, nil)
	s.Require().NoError(err)

	initMsg, err := json.Marshal(oraiswapV3InstantiateMsg{
		ProtocolFee:           250_000_000_000,
		IncentivesFundManager: admin.String(),
	})
	s.Require().NoError(err)

	contractAddr, _, err := keepers.ContractKeeper.Instantiate(
		s.Ctx, codeID, admin, admin, initMsg, "oraiswap-v3-test", nil,
	)
	s.Require().NoError(err)
	return contractAddr
}

func (s *UpgradeTestSuite) oraiswapV3IsPaused(contract sdk.AccAddress) bool {
	q, err := json.Marshal(oraiswapV3IsPausedQuery{})
	s.Require().NoError(err)
	bz, err := s.App.WasmKeeper.QuerySmart(s.Ctx, contract, q)
	s.Require().NoError(err)
	var paused bool
	s.Require().NoError(json.Unmarshal(bz, &paused))
	return paused
}

func (s *UpgradeTestSuite) TestPausePoolV3PausesContract() {
	admin := sdk.AccAddress("oraiswap-v3-admin-addr") // 20 bytes
	pool := s.deployOraiswapV3(admin)

	s.Require().False(s.oraiswapV3IsPaused(pool))

	v05014.BlacklistAddresses = nil
	v05014.AdminContract = admin.String()
	v05014.PausePoolV3 = pool.String()

	s.Ctx = s.Ctx.WithBlockHeight(v05014.ForkHeight)
	keepers := s.App.GetUpgradeKeepers()
	s.Require().NotPanics(func() {
		v05014.RunForkLogic(s.Ctx, &keepers)
	})

	s.Require().True(s.oraiswapV3IsPaused(pool))
}

func (s *UpgradeTestSuite) TestPausePoolV3PanicsOnWrongAdmin() {
	admin := sdk.AccAddress("oraiswap-v3-admin-ad2") // 20 bytes
	wrongAdmin := sdk.AccAddress("oraiswap-v3-wrong-adm") // 20 bytes
	pool := s.deployOraiswapV3(admin)

	v05014.BlacklistAddresses = nil
	v05014.AdminContract = wrongAdmin.String()
	v05014.PausePoolV3 = pool.String()

	s.Ctx = s.Ctx.WithBlockHeight(v05014.ForkHeight)
	keepers := s.App.GetUpgradeKeepers()
	s.Require().Panics(func() {
		v05014.RunForkLogic(s.Ctx, &keepers)
	})
}

func (s *UpgradeTestSuite) TestPausePoolV3PanicKeepsStateUnchanged() {
	admin := sdk.AccAddress("oraiswap-v3-admin-ad3") // 20 bytes
	pool := s.deployOraiswapV3(admin)
	trader := sdk.AccAddress("oraiswap-v3-trader-ad") // 20 bytes
	s.fund(trader, 1_000_000)

	v05014.BlacklistAddresses = nil
	v05014.RevertAddress = []v05014.RevertEntry{
		{Address: trader.String(), Amount: sdkmath.NewInt(2_000_000)}, // panics before pausePoolV3
	}
	v05014.AdminContract = admin.String()
	v05014.PausePoolV3 = pool.String()

	s.Ctx = s.Ctx.WithBlockHeight(v05014.ForkHeight)
	keepers := s.App.GetUpgradeKeepers()
	s.Require().Panics(func() {
		v05014.RunForkLogic(s.Ctx, &keepers)
	})

	s.Require().False(s.oraiswapV3IsPaused(pool))
}
