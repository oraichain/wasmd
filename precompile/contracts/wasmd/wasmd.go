package wasmd

import (
	"embed"
	"fmt"

	pcommon "github.com/CosmWasm/wasmd/precompile/common"
	cmn "github.com/cosmos/evm/precompiles/common"
	"github.com/cosmos/evm/x/vm/core/vm"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

var _ vm.PrecompiledContract = &Precompile{}

// Embed abi json file to the executable binary. Needed when importing as dependency.
//
//go:embed abi.json
var f embed.FS

const (
	WasmdContractAddress = "0x9000000000000000000000000000000000000001"

	// Define the minumum gas required needed to execute
	InstantiateWasmContractRequiredGas = 5_000_000
	ExecuteWasmContractRequiredGas     = 3_000_000
	QueryWasmContractRequiredGas       = 30_000

	InstantiateWasmContractMethod = "instantiate"
	ExecuteWasmContractMethod     = "execute"
	QueryWasmContractMethod       = "query"
)

// Precompile defines the precompiled contract for wasmd.
type Precompile struct {
	cmn.Precompile
	EVMKeeper  pcommon.EVMKeeper
	WasmKeeper pcommon.WasmdKeeper
}

// LoadABI loads the slashing ABI from the embedded abi.json file
// for the slashing precompile.
func LoadABI() (abi.ABI, error) {
	return cmn.LoadABI(f, "abi.json")
}

func NewPrecompile(wasmKeeper pcommon.WasmdKeeper, evmKeeper pcommon.EVMKeeper) (*Precompile, error) {
	abi, err := LoadABI()
	if err != nil {
		return nil, err
	}

	p := &Precompile{
		Precompile: cmn.Precompile{
			ABI: abi,
		},
		EVMKeeper:  evmKeeper,
		WasmKeeper: wasmKeeper,
	}

	// SetAddress defines the address of the bank compile contract.
	p.SetAddress(common.HexToAddress(WasmdContractAddress))

	return p, nil
}

func (p Precompile) Address() common.Address {
	return p.Precompile.Address()
}

// RequiredGas calculates the precompiled contract's base gas rate.
func (p Precompile) RequiredGas(input []byte) uint64 {
	if len(input) < 4 {
		return 0
	}
	methodID := input[:4]

	method, err := p.MethodById(methodID)
	if err != nil {
		return 0
	}

	switch method.Name {
	case InstantiateWasmContractMethod:
		return InstantiateWasmContractRequiredGas
	case ExecuteWasmContractMethod:
		return ExecuteWasmContractRequiredGas
	case QueryWasmContractMethod:
		return QueryWasmContractRequiredGas
	}

	return 0
}

// Run executes the precompiled contract bank query methods defined in the ABI.
func (p Precompile) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) (bz []byte, err error) {
	ctx, stateDB, snapshot, method, initialGas, args, err := p.RunSetup(evm, contract, readOnly, p.IsTransaction)
	if err != nil {
		return nil, err
	}

	// This handles any out of gas errors that may occur during the execution of a precompile query.
	// It avoids panics and returns the out of gas error so the EVM can continue gracefully.
	defer cmn.HandleGasError(ctx, contract, initialGas, &err)()

	switch method.Name {
	case InstantiateWasmContractMethod:
		bz, err = p.Instantiate(ctx, contract, method, args)
	case ExecuteWasmContractMethod:
		bz, err = p.Execute(ctx, contract, method, args)
	case QueryWasmContractMethod:
		bz, err = p.Query(ctx, contract, method, args)
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

// IsTransaction checks if the given method name corresponds to a transaction or query.
// It returns false since all bank methods are queries.
func (Precompile) IsTransaction(method *abi.Method) bool {
	switch method.Name {
	case InstantiateWasmContractMethod,
		ExecuteWasmContractMethod:
		return true
	default:
		return false
	}
}
