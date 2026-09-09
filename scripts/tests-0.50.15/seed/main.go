// Command seed emits a single gov v1 Proposal JSON object carrying an EVM message
// (/cosmos.evm.vm.v1.MsgUpdateParams). 01-init.sh splices it into
// data/oraid/config/genesis.json under .app_state.gov.proposals so the fork test
// starts from a chain state that already contains a "historical" EVM proposal —
// the same situation Oraichain mainnet is in with proposal #316 after the EVM
// module was soft-removed in v0.50.14.
//
// Usage: go run ./scripts/tests-0.50.15/seed [proposalID]
package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	evmtypes "github.com/cosmos/evm/x/vm/types"
)

func main() {
	id := uint64(1)
	if len(os.Args) > 1 {
		v, err := strconv.ParseUint(os.Args[1], 10, 64)
		if err != nil {
			panic(fmt.Errorf("bad proposal id %q: %w", os.Args[1], err))
		}
		id = v
	}

	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount("orai", "oraipub")

	// Same registration the decode-only stub (app/evmlegacy) performs.
	reg := codectypes.NewInterfaceRegistry()
	reg.RegisterImplementations((*sdk.Msg)(nil), &evmtypes.MsgUpdateParams{})
	cdc := codec.NewProtoCodec(reg)

	authority := authtypes.NewModuleAddress("gov").String()
	anyMsg, err := codectypes.NewAnyWithValue(&evmtypes.MsgUpdateParams{
		Authority: authority,
		Params:    evmtypes.DefaultParams(),
	})
	if err != nil {
		panic(err)
	}

	submit := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)
	depEnd := submit.Add(48 * time.Hour)
	voteEnd := submit.Add(72 * time.Hour)

	prop := govv1.Proposal{
		Id:       id,
		Messages: []*codectypes.Any{anyMsg},
		Status:   govv1.StatusPassed,
		FinalTallyResult: &govv1.TallyResult{
			YesCount:        "0",
			AbstainCount:    "0",
			NoCount:         "0",
			NoWithVetoCount: "0",
		},
		SubmitTime:      &submit,
		DepositEndTime:  &depEnd,
		TotalDeposit:    nil,
		VotingStartTime: &submit,
		VotingEndTime:   &voteEnd,
		Metadata:        "",
		Title:           "legacy evm MsgUpdateParams",
		Summary:         "historical gov proposal carrying /cosmos.evm.vm.v1.MsgUpdateParams (mirrors mainnet proposal 316)",
		Proposer:        authority,
	}

	out, err := cdc.MarshalJSON(&prop)
	if err != nil {
		panic(err)
	}
	if _, err := os.Stdout.Write(out); err != nil {
		panic(err)
	}
	fmt.Println()
}
