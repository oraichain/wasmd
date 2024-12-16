package keeper

import (
	"encoding/json"
	"os"
	"testing"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

func TestSelectAuthorizationPolicy(t *testing.T) {
	myGovAuthority := RandomAccountAddress(t)
	m := msgServer{keeper: &Keeper{
		propagateGovAuthorization: map[types.AuthorizationPolicyAction]struct{}{
			types.AuthZActionMigrateContract: {},
			types.AuthZActionInstantiate:     {},
		},
		authority: myGovAuthority.String(),
	}}

	ms := store.NewCommitMultiStore(dbm.NewMemDB(), log.NewTestLogger(t), storemetrics.NewNoOpMetrics())
	ctx := sdk.NewContext(ms, tmproto.Header{}, false, log.NewNopLogger())

	specs := map[string]struct {
		ctx   sdk.Context
		actor sdk.AccAddress
		exp   types.AuthorizationPolicy
	}{
		"always gov policy for gov authority sender": {
			ctx:   types.WithSubMsgAuthzPolicy(ctx, NewPartialGovAuthorizationPolicy(nil, types.AuthZActionMigrateContract)),
			actor: myGovAuthority,
			exp:   NewGovAuthorizationPolicy(types.AuthZActionMigrateContract, types.AuthZActionInstantiate),
		},
		"pick from context when set": {
			ctx:   types.WithSubMsgAuthzPolicy(ctx, NewPartialGovAuthorizationPolicy(nil, types.AuthZActionMigrateContract)),
			actor: RandomAccountAddress(t),
			exp:   NewPartialGovAuthorizationPolicy(nil, types.AuthZActionMigrateContract),
		},
		"fallback to default policy": {
			ctx:   ctx,
			actor: RandomAccountAddress(t),
			exp:   DefaultAuthorizationPolicy{},
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			got := m.selectAuthorizationPolicy(spec.ctx, spec.actor.String())
			assert.Equal(t, spec.exp, got)
		})
	}
}

func TestSetGaslessAndUnsetGasLessProposal(t *testing.T) {
	ctx, keepers := CreateTestInput(t, false, AvailableCapabilities)
	wasmKeeper := keepers.WasmKeeper
	govKeeper := keepers.GovKeeper

	myAddress := DeterministicAccountAddress(t, 1)

	wasmKeeper.SetParams(ctx, types.Params{
		CodeUploadAccess:             types.AllowEverybody,
		InstantiateDefaultPermission: types.AccessTypeEverybody,
	})

	wasmCode, err := os.ReadFile("./testdata/hackatom.wasm")
	require.NoError(t, err)

	codeInfo := types.CodeInfoFixture(types.WithSHA256CodeHash(wasmCode), func(codeInfo *types.CodeInfo) {
		codeInfo.Creator = sdk.AccAddress(myAddress).String()
	})
	err = wasmKeeper.importCode(ctx, 1, codeInfo, wasmCode)
	require.NoError(t, err)

	// instantiate contract
	_, bob := keyPubAddr()
	initMsg := HackatomExampleInitMsg{
		Verifier:    myAddress,
		Beneficiary: bob,
	}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)
	contractAddress, _, err := wasmKeeper.instantiate(ctx, 1, myAddress, myAddress, initMsgBz, "labels", nil, wasmKeeper.ClassicAddressGenerator(), DefaultAuthorizationPolicy{})
	require.NoError(t, err)

	// Test SetGasLess
	// store proposal
	em := sdk.NewEventManager()
	msgSetGasLessProposal := &types.MsgSetGaslessContracts{
		Authority: wasmKeeper.GetAuthority(),
		Contracts: []string{contractAddress.String()},
	}
	storedProposal, err := govKeeper.SubmitProposal(ctx, []sdk.Msg{msgSetGasLessProposal}, "metadata", "title", "sumary", myAddress, true)
	require.NoError(t, err)

	// execute proposal
	msgs, err := sdktx.GetMsgs(storedProposal.Messages, "sdk.MsgProposal")
	require.NoError(t, err)

	handler := govKeeper.Router().Handler(msgs[0])
	result, err := handler(ctx.WithEventManager(em), msgs[0])
	require.NoError(t, err)
	require.NotEmpty(t, result)

	// check store
	isGasLess := wasmKeeper.IsGasless(ctx, contractAddress)
	require.True(t, isGasLess)

	// Test UnsetGasLess
	msgUnsetGasLessProposal := &types.MsgUnsetGaslessContracts{
		Authority: wasmKeeper.GetAuthority(),
		Contracts: []string{contractAddress.String()},
	}
	storedProposal, err = govKeeper.SubmitProposal(ctx, []sdk.Msg{msgUnsetGasLessProposal}, "metadata", "title", "sumary", myAddress, true)
	require.NoError(t, err)

	// execute proposal
	msgs, err = sdktx.GetMsgs(storedProposal.Messages, "sdk.MsgProposal")
	require.NoError(t, err)

	handler = govKeeper.Router().Handler(msgs[0])
	result, err = handler(ctx.WithEventManager(em), msgs[0])
	require.NoError(t, err)
	require.NotEmpty(t, result)

	// check store
	isGasLess = wasmKeeper.IsGasless(ctx, contractAddress)
	require.False(t, isGasLess)
}
