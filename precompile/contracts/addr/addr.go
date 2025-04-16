package addr

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

// Singleton StatefulPrecompiledContract.
var (
	// RawABI contains the raw ABI of addr contract.
	//go:embed abi.json
	RawABI string

	ABI = contract.MustParseABI(RawABI)
)

const (
	AddrContractAddress = "0x9000000000000000000000000000000000000003"

	// Define the minumum gas required needed for each method
	// TODO: need to re-define gas required here
	AssociateMethodRequiredGas        = 5_000_000
	AssociatePubKeyMethodRequiredGas  = 5_000_000
	GetCosmosAddressMethodRequiredGas = 30_000
	GetEvmAddressMethodRequiredGas    = 30_000

	// Execute methods
	AssociateMethod       = "associate"
	AssociatePubKeyMethod = "associatePubKey"

	// Query methods
	GetCosmosAddressMethod = "getCosmosAddr"
	GetEvmAddressMethod    = "getEvmAddr"
)

// Precompile defines the precompiled contract for addr.
type Precompile struct {
	cmn.Precompile
	EVMKeeper pcommon.EVMKeeper
}

func NewPrecompile(evmKeeper pcommon.EVMKeeper) (*Precompile, error) {
	p := &Precompile{
		Precompile: cmn.Precompile{
			ABI: ABI,
		},
		EVMKeeper: evmKeeper,
	}

	// SetAddress defines the address of the addr compile contract.
	p.SetAddress(common.HexToAddress(AddrContractAddress))

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
	case AssociateMethod:
		return AssociateMethodRequiredGas
	case AssociatePubKeyMethod:
		return AssociatePubKeyMethodRequiredGas
	case GetCosmosAddressMethod:
		return GetCosmosAddressMethodRequiredGas
	case GetEvmAddressMethod:
		return GetEvmAddressMethodRequiredGas
	}

	return 0
}

// Run executes the precompiled contract addr methods defined in the ABI.
func (p Precompile) Run(
	evm *vm.EVM,
	contract *vm.Contract,
	readOnly bool,
) (bz []byte, err error) {
	ctx, stateDB, snapshot, method, initialGas, args, err := p.RunSetup(evm, contract, readOnly, p.IsTransaction)
	if err != nil {
		return nil, err
	}

	// This handles any out of gas errors that may occur during the execution of a precompile.
	// It avoids panics and returns the out of gas error so the EVM can continue gracefully.
	defer cmn.HandleGasError(ctx, contract, initialGas, &err)()

	switch method.Name {
	case AssociateMethod:
		bz, err = p.Associate(ctx, contract, method, args)
		break
	case AssociatePubKeyMethod:
		bz, err = p.AssociatePubKey(ctx, contract, method, args)
		break
	case GetCosmosAddressMethod:
		bz, err = p.GetCosmosAddr(ctx, contract, method, args)
		break
	case GetEvmAddressMethod:
		bz, err = p.GetEvmAddr(ctx, contract, method, args)
		break
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
func (Precompile) IsTransaction(method *abi.Method) bool {
	switch method.Name {
	case AssociateMethod:
	case AssociatePubKeyMethod:
		return true
	default:
		return false
	}

	return false
}
