package json

import (
	_ "embed"
	"fmt"

	pcommon "github.com/CosmWasm/wasmd/precompile/common"
	cmn "github.com/cosmos/evm/precompiles/common"
	"github.com/cosmos/evm/x/vm/core/vm"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/precompile/contract"
)

var _ vm.PrecompiledContract = &Precompile{}

// Singleton StatefulPrecompiledContract.
var (
	// RawABI contains the raw ABI of wasmd contract.
	//go:embed abi.json
	RawABI string

	ABI = contract.MustParseABI(RawABI)
)

const (
	// @TODO: These values are placeholders and should be replaced with the actual gas values
	GasExtractAsBytes     = 30_000
	GasExtractAsBytesList = 30_000
	GasExtractAsUint256   = 30_000

	ExtractAsBytesMethod     = "extractAsBytes"
	ExtractAsBytesListMethod = "extractAsBytesList"
	ExtractAsUint256Method   = "extractAsUint256"
)

type Precompile struct {
	cmn.Precompile
}

func NewPrecompile() (*Precompile, error) {
	p := &Precompile{
		Precompile: cmn.Precompile{
			ABI: ABI,
		},
	}

	// SetAddress defines the address of the bank precompile contract.
	p.SetAddress(common.HexToAddress(pcommon.JsonContractAddress))

	return p, nil
}

func (p Precompile) Address() common.Address {
	return p.Precompile.Address()
}

// RequiredGas calculates the precompiled contract's base gas rate.
func (p Precompile) RequiredGas(input []byte) uint64 {
	// NOTE: This check avoid panicking when trying to decode the method ID
	if len(input) < 4 {
		return 0
	}

	methodID := input[:4]

	method, err := p.MethodById(methodID)
	if err != nil {
		// This should never happen since this method is going to fail during Run
		return 0
	}

	switch method.Name {
	case ExtractAsBytesMethod:
		return GasExtractAsBytes
	case ExtractAsBytesListMethod:
		return GasExtractAsBytesList
	case ExtractAsUint256Method:
		return GasExtractAsUint256
	default:
		return 0
	}
}

func (p Precompile) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) (bz []byte, err error) {
	ctx, stateDB, snapshot, method, initialGas, args, err := p.RunSetup(evm, contract, readOnly, p.IsTransaction)
	if err != nil {
		return nil, err
	}

	// This handles any out of gas errors that may occur during the execution of a precompile query.
	// It avoids panics and returns the out of gas error so the EVM can continue gracefully.
	defer cmn.HandleGasError(ctx, contract, initialGas, &err)()

	switch method.Name {
	case ExtractAsBytesMethod:
		bz, err = p.ExtractAsBytes(ctx, contract, method, args)
	case ExtractAsBytesListMethod:
		bz, err = p.ExtractAsBytesList(ctx, contract, method, args)
	case ExtractAsUint256Method:
		bz, err = p.ExtractAsUint256(ctx, contract, method, args)
	default:
		return nil, fmt.Errorf(cmn.ErrUnknownMethod, method.Name)
	}

	if err != nil {
		return nil, err
	}

	cost := ctx.GasMeter().GasConsumed() - initialGas

	if !contract.UseGas(cost) {
		return nil, vm.ErrOutOfGas
	}

	if err := p.AddJournalEntries(stateDB, snapshot); err != nil {
		return nil, err
	}

	return bz, nil
}

func (p Precompile) IsTransaction(method *abi.Method) bool {
	// All methods in this precompile are read-only (queries)
	return false
}
