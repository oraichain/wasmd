package wasm

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/CosmWasm/wasmd/x/wasm/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
)

func TestInitGenesis(t *testing.T) {
	data := setupTest(t)

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	topUp := sdk.NewCoins(sdk.NewInt64Coin("denom", 5000))
	creator := data.faucet.NewFundedRandomAccount(data.ctx, deposit.Add(deposit...)...)
	fred := data.faucet.NewFundedRandomAccount(data.ctx, topUp...)

	totalCodeIds := 13
	totalContracts := 1131

	h := data.module.Route().Handler()
	q := data.module.LegacyQuerierHandler(nil)

	msg := MsgStoreCode{
		Sender:       creator.String(),
		WASMByteCode: testContract,
	}
	err := msg.ValidateBasic()
	require.NoError(t, err)

	for i := 1; i <= totalCodeIds; i++ {
		res, err := h(data.ctx, &msg)
		require.NoError(t, err)
		assertStoreCodeResponse(t, res.Data, uint64(i))
	}

	_, _, bob := keyPubAddr()
	initMsg := initMsg{
		Verifier:    fred,
		Beneficiary: bob,
	}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)

	initCmd := MsgInstantiateContract{
		Sender: creator.String(),
		CodeID: firstCodeID,
		Msg:    initMsgBz,
		Funds:  sdk.NewCoins(),
		Label:  "testing",
	}

	contractAddrs := []string{}
	for i := 1; i <= totalContracts; i++ {
		res, err := h(data.ctx, &initCmd)
		require.NoError(t, err)
		contractBech32Addr := parseInitResponse(t, res.Data)
		contractAddrs = append(contractAddrs, contractBech32Addr)

		execCmd := MsgExecuteContract{
			Sender:   fred.String(),
			Contract: contractBech32Addr,
			Msg:      []byte(`{"release":{}}`),
			Funds:    sdk.NewCoins(),
		}
		res, err = h(data.ctx, &execCmd)
		require.NoError(t, err)
		// from https://github.com/CosmWasm/cosmwasm/blob/master/contracts/hackatom/src/contract.rs#L167
		assertExecuteResponse(t, res.Data, []byte{0xf0, 0x0b, 0xaa})
		assertContractInfo(t, q, data.ctx, contractBech32Addr, 1, creator)
		assertContractState(t, q, data.ctx, contractBech32Addr, state{
			Verifier:    fred.String(),
			Beneficiary: bob.String(),
			Funder:      creator.String(),
		})
	}
	// ensure all contract state is as after init
	assertCodeList(t, q, data.ctx, totalCodeIds)
	assertCodeBytes(t, q, data.ctx, 1, testContract)
	assertContractList(t, q, data.ctx, 1, contractAddrs)
	contractBech32Addr := contractAddrs[0]

	// export into genstate
	genState := ExportGenesis(data.ctx, &data.keeper)
	fmt.Println("gen state: ", len(genState.Contracts))

	// create new app to import genstate into
	newData := setupTest(t)
	q2 := newData.module.LegacyQuerierHandler(nil)

	// initialize new app with genstate
	start := time.Now() // Record the start time
	InitGenesis(newData.ctx, &newData.keeper, genState)
	duration := time.Since(start) // Calculate the elapsed time
	fmt.Printf("Process ran for %s\n", duration)

	// run same checks again on newdata, to make sure it was reinitialized correctly
	assertCodeList(t, q2, newData.ctx, totalCodeIds)
	assertCodeBytes(t, q2, newData.ctx, 1, testContract)

	assertContractList(t, q2, newData.ctx, 1, contractAddrs)
	assertContractInfo(t, q2, newData.ctx, contractBech32Addr, 1, creator)
	// assertContractState(t, q2, newData.ctx, contractBech32Addr, state{
	// 	Verifier:    fred.String(),
	// 	Beneficiary: bob.String(),
	// 	Funder:      creator.String(),
	// })

	errCount := 0
	for _, contract := range contractAddrs {
		t.Helper()
		path := []string{QueryGetContractState, contract, keeper.QueryMethodContractStateAll}
		bz, sdkerr := q2(newData.ctx, path, abci.RequestQuery{})
		require.NoError(t, sdkerr)

		var modelRes []Model
		err = json.Unmarshal(bz, &modelRes)
		require.NoError(t, err)
		if len(modelRes) == 0 {
			errCount++
			continue
		}
		require.Equal(t, 1, len(modelRes), "#v", modelRes)
		require.Equal(t, []byte("config"), []byte(modelRes[0].Key))

		expectedBz, err := json.Marshal(state{
			Verifier:    fred.String(),
			Beneficiary: bob.String(),
			Funder:      creator.String(),
		})
		require.NoError(t, err)
		assert.Equal(t, expectedBz, modelRes[0].Value)
	}
	require.Equal(t, errCount, 0)
	fmt.Println("err count: ", errCount)

}
