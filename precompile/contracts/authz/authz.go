package authz

import (
	"errors"
	"fmt"
	"math/big"

	pcommon "github.com/CosmWasm/wasmd/precompile/common"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/precompile/contract"
)

// Singleton StatefulPrecompiledContract.
var (
	// RawABI contains the raw ABI of wasmd contract.
	// go:embed abi.json
	RawABI string

	ABI = contract.MustParseABI(RawABI)
)

const (
	ApproveMethod = "aprrove"
)

type PrecompileExecutor struct {
	evmKeeper   pcommon.EVMKeeper
	authzKeeper pcommon.AuthzKeeper
}

// NewContract returns a new authz stateful precompiled contract.
//
//	This contract is used for testing purposes only and should not be used on public chains.
//	The functions of this contract (once implemented), will be used to exercise and test the various aspects of
//	the EVM such as gas usage, argument parsing, events, etc. The specific operations tested under this contract are
//	still to be determined.
func NewContract(evmKeeper pcommon.EVMKeeper, authzKeeper pcommon.AuthzKeeper) contract.StatefulPrecompiledContract {
	executor := &PrecompileExecutor{
		evmKeeper:   evmKeeper,
		authzKeeper: authzKeeper,
	}

	functions := []*contract.StatefulPrecompileFunction{
		contract.NewStatefulPrecompileFunction(
			ABI.Methods[ApproveMethod].ID,
			executor.approve,
		),
	}

	// Construct the contract with functions.
	precompile, err := contract.NewStatefulPrecompileContract(functions)
	if err != nil {
		panic(fmt.Sprintf("failed to instantiate authz precompile: %s", err.Error()))
	}

	return precompile
}

func (p PrecompileExecutor) approve(
	accessibleState contract.AccessibleState,
	caller common.Address,
	callingContract common.Address,
	packedInput []byte,
	suppliedGas uint64,
	readOnly bool,
	value *big.Int,
) (ret []byte, remainingGas uint64, rerr error) {
	_, rerr = pcommon.GetPrecompileCtx(accessibleState)
	if rerr != nil {
		return
	}

	defer func() {
		if err := recover(); err != nil {
			ret = nil
			remainingGas = 0
			rerr = fmt.Errorf("%s", err)
			return

		}
	}()

	method := ABI.Methods[ApproveMethod]
	args, err := method.Inputs.Unpack(packedInput)
	if err != nil {
		rerr = err
		return
	}

	if readOnly {
		rerr = errors.New("cannot call approve from staticcall")
		return
	}

	if err := pcommon.ValidateNonPayable(value); err != nil {
		rerr = err
		return
	}

	// TODO: need to check args length again
	if err := pcommon.ValidateArgsLength(args, 3); err != nil {
		rerr = err
		return
	}

	// TODO: need to implement authz logic here
	return
}
